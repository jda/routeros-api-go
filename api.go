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

	"joi.com.br/mikrotik-go/proto"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// Reply has all lines from a reply. They must all have the same .tag value.
type Reply struct {
	Re   []*proto.Sentence
	Done *proto.Sentence
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
	Address   string
	Username  string
	Password  string
	TLSConfig *tls.Config
	conn      net.Conn
	r         proto.Reader
	w         *proto.Writer
	async     bool
	nextTag   int64
	tags      map[string]*AsyncReply
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
	c.r = proto.NewReader(c.conn)
	c.w = proto.NewWriter(c.conn)

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
	c.w.BeginSentence()
	c.w.WriteWord(command)

	// Set property list if present
	if len(q.Proplist) > 0 {
		c.w.Printf("=.proplist=%s", strings.Join(q.Proplist, ","))
	}

	// send params if we got them
	if len(q.Pairs) > 0 {
		for _, v := range q.Pairs {
			c.w.Printf("?%s%s=%s", v.Op, v.Key, v.Value)
		}

		if q.Op != "" {
			c.w.Printf("?#%s", q.Op)
		}
	}

	return c.endCommand()
}

func (c *Client) Call(command string, params []Pair) (*Reply, error) {
	c.w.BeginSentence()
	c.w.WriteWord(command)

	// send params if we got them
	if len(params) > 0 {
		for _, v := range params {
			c.w.Printf("=%s=%s", v.Key, v.Value)
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
	err := c.w.EndSentence()
	if err != nil {
		return nil, err
	}
	return c.readReply()
}

func (c *Client) endCommandAsync() (*Reply, error) {
	a := c.newAsyncReply()
	c.w.Printf(".tag=%s", a.Tag)

	c.Lock()
	err := c.w.EndSentence()
	if err != nil {
		c.Unlock()
		return nil, err
	}
	c.tags[a.Tag] = a
	c.Unlock()

	<-a.C

	c.Lock()
	delete(c.tags, a.Tag)
	c.Unlock()

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
