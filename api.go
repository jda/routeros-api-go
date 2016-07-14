package routeros

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"joi.com.br/mikrotik-go/sentence"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// A reply can contain multiple pairs. A pair is a string key->value.
// A reply can also contain subpairs, that is, a array of pair arrays.
type Reply struct {
	Re   []*sentence.Sentence
	Done *sentence.Sentence
}

func (r *Reply) String() string {
	b := &bytes.Buffer{}
	for _, re := range r.Re {
		fmt.Fprintf(b, "%s\n", re)
	}
	fmt.Fprintf(b, "%s", r.Done)
	return b.String()
}

// Client is a RouterOS API client.
type Client struct {
	Address        string
	Username       string
	Password       string
	TLSConfig      *tls.Config
	async          bool
	conn           net.Conn
	sentenceReader sentence.Reader
	sentenceWriter sentence.Writer
	mu             sync.Mutex
}

// Pair is a Key-Value pair for RouterOS Attribute, Query, and Reply words
// use slices of pairs instead of map because we care about order
type Pair struct {
	Key   string
	Value string
	// Op is used for Query words to signify logical operations
	// valid operators are -, =, <, >
	// see http://wiki.mikrotik.com/wiki/Manual:API#Queries for details.
	Op string
}

type Query struct {
	Pairs    []Pair
	Op       string
	Proplist []string
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Connect() error {
	var err error
	if c.TLSConfig != nil {
		c.conn, err = tls.Dial("tcp", c.Address, c.TLSConfig)
	} else {
		c.conn, err = net.Dial("tcp", c.Address)
	}
	if err != nil {
		return err
	}
	c.sentenceReader = sentence.NewReader(c.conn)
	c.sentenceWriter = sentence.NewWriter(c.conn)

	// try to log in
	res, err := c.Call("/login", nil)
	if err != nil {
		return err
	}

	// handle challenge/response
	challengeEnc, ok := res.Done.Map["ret"]
	if !ok {
		return errors.New("Didn't get challenge from ROS")
	}
	challenge, err := hex.DecodeString(challengeEnc)
	if err != nil {
		return err
	}
	h := md5.New()
	io.WriteString(h, "\000")
	io.WriteString(h, c.Password)
	h.Write(challenge)
	resp := fmt.Sprintf("00%x", h.Sum(nil))

	// try to log in again with challenge/response
	res, err = c.Call("/login", []Pair{
		{Key: "name", Value: c.Username},
		{Key: "response", Value: resp},
	})
	if err != nil {
		return err
	}

	if len(res.Done.List) > 0 {
		return fmt.Errorf("Unexpected result on login: %#q", res.Done.List)
	}

	return nil
}

func (c *Client) Query(command string, q Query) (*Reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	w := c.sentenceWriter
	w.WriteString(command)

	// Set property list if present
	if len(q.Proplist) > 0 {
		proplist := fmt.Sprintf("=.proplist=%s", strings.Join(q.Proplist, ","))
		w.WriteString(proplist)
	}

	// send params if we got them
	if len(q.Pairs) > 0 {
		for _, v := range q.Pairs {
			word := fmt.Sprintf("?%s%s=%s", v.Op, v.Key, v.Value)
			w.WriteString(word)
		}

		if q.Op != "" {
			word := fmt.Sprintf("?#%s", q.Op)
			w.WriteString(word)
		}
	}

	// send terminator
	w.WriteString("")
	if w.Err() != nil {
		return nil, w.Err()
	}

	res, err := c.readReply()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) Call(command string, params []Pair) (*Reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	w := c.sentenceWriter
	w.WriteString(command)

	// send params if we got them
	if len(params) > 0 {
		for _, v := range params {
			word := fmt.Sprintf("=%s=%s", v.Key, v.Value)
			w.WriteString(word)
		}
	}

	// send terminator
	w.WriteString("")
	if w.Err() != nil {
		return nil, w.Err()
	}

	res, err := c.readReply()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Loop starts asynchronous mode.
func (c *Client) Loop() error {
	c.async = true
	defer func() { c.async = false }()

	for {
		reply, err := c.readReply()
		if err != nil {
			return err
		}
		_ = reply
	}
}
