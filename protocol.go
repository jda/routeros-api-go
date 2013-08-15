// Package routeros provides a programmatic interface to to the Mikrotik RouterOS API
package routeros

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jcelliott/lumber"
	"io"
	"net"
	"strconv"
	"strings"
)

// Client is a RouterOS API client.
type Client struct {
	// Network Address.
	// E.g. "10.0.0.1:8728" or "router.example.com:8728"
	address  string
	user     string
	password string
	logger   *lumber.Logger
	debug    bool     // debug logging enabled
	ready    bool     // Ready for work (login ok and connection not terminated)
	conn     net.Conn // Connection to pass around
}

// Pair is a Key-Value pair for RouterOS Attribute, Query, and Reply words
// use slices of pairs instead of map because we care about order
type Pair struct {
	Key   string
	Value string
	// Op is used for Query words to signify logical operations
	// valid operators are -, =, <, >
	// see http://wiki.mikrotik.com/wiki/Manual:API#Queries for details.
	Op    string
}

type Query struct {
	Pairs    []Pair
	Op       string
	Proplist []string
}

func NewPair(key string, value string) *Pair {
	p := new(Pair)
	p.Key = key
	p.Value = value
	return p
}

// Get value for a specific key
func GetPairValue(p []Pair, key string) (string, error) {
	for _, v := range p {
		if v.Key == key {
			return v.Value, nil
		}
	}
	return "", errors.New("key not found")
}

// Create a new instance of the RouterOS API client
func NewRouterOSClient(address string) (*Client, error) {
	// basic validation of host address
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	var c Client
	c.address = address

	return &c, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Connect(user string, password string) error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	// stash conn in instance
	c.conn = conn

	// try to log in
	res, err := c.Call("/login", nil)
	if err != nil {
		return err
	}

	// handle challenge/response
	challengeEnc, err := GetPairValue(res, "ret")
	if err != nil {
		return errors.New("Didn't get challenge from ROS")
	}
	challenge, err := hex.DecodeString(challengeEnc)
	if err != nil {
		return err
	}
	h := md5.New()
	io.WriteString(h, "\000")
	io.WriteString(h, password)
	h.Write(challenge)
	resp := fmt.Sprintf("00%x", h.Sum(nil))
	var loginParams []Pair
	loginParams = append(loginParams, *NewPair("name", password))
	loginParams = append(loginParams, *NewPair("response", resp))

	// try to log in again with challenge/response
	res, err = c.Call("/login", loginParams)
	if err != nil {
		return err
	}
	fmt.Println(res)

	return nil
}

// Encode and send a single line
func (c *Client) send(word string) error {
	bword := []byte(word)
	prefix := prefixlen(len(bword))

	_, err := c.conn.Write(prefix.Bytes())
	if err != nil {
		return err
	}

	_, err = c.conn.Write(bword)
	if err != nil {
		return err
	}

	return nil
}

// Listen for reply
func (c *Client) receive() ([]Pair, error) {
	var pairs []Pair

	for {
		length := c.getlen()
		if length == 0 {
			break
		}
		inbuf := make([]byte, length)
		c.conn.Read(inbuf)
		word := string(inbuf)

		if word == "!done" {
			continue
		}

		if strings.Contains(word, "=") {
			var p Pair
			parts := strings.SplitN(word, "=", 3)
			p.Key = parts[1]
			p.Value = parts[2]
			pairs = append(pairs, p)
		}

	}

	return pairs, nil
}

func (c *Client) Query(command string, q Query) ([]Pair, error) {
	err := c.send(command)
	if err != nil {
		return nil, err
	}

	// Set property list if present
	if len(q.Proplist) > 0 {
		proplist := fmt.Sprintf("=.proplist=%s", strings.Join(q.Proplist, ","))
		err = c.send(proplist)
		if err != nil {
			return nil, err
		}
	}

	// send params if we got them
	if len(q.Pairs) > 0 {
		for _, v := range q.Pairs {
			word := fmt.Sprintf("?%s%s=%s", v.Op, v.Key, v.Value)
			c.send(word)
		}

		if q.Op != "" {
			word := fmt.Sprintf("?#%s", q.Op)
			c.send(word)
		}
	}

	// send terminator
	err = c.send("")
	if err != nil {
		return nil, err
	}

	res, err := c.receive()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) Call(command string, params []Pair) ([]Pair, error) {
	err := c.send(command)
	if err != nil {
		return nil, err
	}

	// send params if we got them
	if len(params) > 0 {
		for _, v := range params {
			word := fmt.Sprintf("=%s=%s", v.Key, v.Value)
			c.send(word)
		}
	}

	// send terminator
	err = c.send("")
	if err != nil {
		return nil, err
	}

	res, err := c.receive()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Enabled logging by providing a Logger instance from Lumber.
func (c *Client) Logging(l *lumber.Logger, level string) {
	c.logger = l
	if level == "DEBUG" {
		c.debug = true
	}
}

// Get just one byte because MT's size prefix is overoptimized
func (c *Client) getone() int {
	charlet := make([]byte, 1)
	c.conn.Read(charlet)
	numlet := int(charlet[0])
	return numlet
}

// Decode RouterOS API Word Size Prefix / Figure out how much to read
// TODO: based on MT Docs. Look for way to make this cleaner later
func (client *Client) getlen() int64 {
	c := int64(client.getone())

	if (c & 0x80) == 0x00 {

	} else if (c & 0xC0) == 0x80 {
		c &= ^0xC0
		c <<= 8
		c += int64(client.getone())
	} else if (c & 0xE0) == 0xC0 {
		c &= ^0xE0
		c <<= 8
		c += int64(client.getone())
		c <<= 8
		c += int64(client.getone())
	} else if (c & 0xF0) == 0xE0 {
		c &= ^0xF0
		c <<= 8
		c += int64(client.getone())
		c <<= 8
		c += int64(client.getone())
		c <<= 8
		c += int64(client.getone())
	} else if (c & 0xF8) == 0xF0 {
		c = int64(client.getone())
		c <<= 8
		c += int64(client.getone())
		c <<= 8
		c += int64(client.getone())
		c <<= 8
		c += int64(client.getone())
	}

	return c
}

// Calculate RouterOS API Word Size Prefix
// TODO: based on MT Docs. Look for way to make this cleaner later
func prefixlen(l int) bytes.Buffer {
	var b bytes.Buffer

	if l < 0x80 {
		b.Write([]byte(string(l)))
	} else if l < 0x4000 {
		l |= 0x8000
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else if l < 0x200000 {
		l |= 0xC00000
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else if l < 0x10000000 {
		l |= 0xE0000000
		b.Write([]byte(strconv.Itoa((l >> 24) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else {
		b.Write([]byte(strconv.Itoa(0xF0)))
		b.Write([]byte(strconv.Itoa((l >> 24) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	}

	return b
}
