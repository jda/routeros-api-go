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

// Reply has all lines from a reply. They must all have the same .tag value.
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

type AsyncReply struct {
	*Reply
	Tag string
	Err error
	C   chan struct{}
}

// Client is a RouterOS API client.
type Client struct {
	Address        string
	Username       string
	Password       string
	TLSConfig      *tls.Config
	conn           net.Conn
	sentenceReader sentence.Reader
	sentenceWriter sentence.Writer
	async          bool
	nextTag        int64
	tags           map[string]*AsyncReply
	sync.Mutex
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
	c.Lock()
	defer c.Unlock()

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

	return c.endCommand()
}

func (c *Client) Call(command string, params []Pair) (*Reply, error) {
	c.Lock()
	defer c.Unlock()

	w := c.sentenceWriter
	w.WriteString(command)

	// send params if we got them
	if len(params) > 0 {
		for _, v := range params {
			word := fmt.Sprintf("=%s=%s", v.Key, v.Value)
			w.WriteString(word)
		}
	}

	return c.endCommand()
}

func (c *Client) endCommand() (*Reply, error) {
	if c.async {
		return c.endCommandAsync()
	}
	return c.endCommandSync()
}

func (c *Client) endCommandSync() (*Reply, error) {
	w := c.sentenceWriter
	w.WriteString("")
	if w.Err() != nil {
		return nil, w.Err()
	}
	return c.readReply()
}

func (c *Client) endCommandAsync() (*Reply, error) {
	a := c.newAsyncReply()
	w := c.sentenceWriter
	w.WriteString(fmt.Sprintf(".tag=%s", a.Tag))
	c.Lock()
	w.WriteString("")
	if w.Err() != nil {
		c.Unlock()
		return nil, w.Err()
	}
	c.addAsyncReply(a)
	c.Unlock()
	<-a.C
	return a.Reply, a.Err
}

func (c *Client) newAsyncReply() *AsyncReply {
	c.nextTag++
	return &AsyncReply{
		Reply: &Reply{},
		Tag:   fmt.Sprintf("%d", c.nextTag),
		C:     make(chan struct{}),
	}
}

func (c *Client) addAsyncReply(a *AsyncReply) {
	go func() {
		<-a.C
		c.Lock()
		delete(c.tags, a.Tag)
		c.Unlock()
	}()
	c.tags[a.Tag] = a
}
