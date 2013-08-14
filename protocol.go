// Package routeros provides a Client interface to to the Mikrotik RouterOS API
package routeros

import (
	"bytes"
	"fmt"
	"github.com/jcelliott/lumber"
	"net"
	"strconv"
	"strings"
	"errors"
	"crypto/md5"
	"io"
)

// A Client is a RouterOS API client.
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

// use slices of pairs instead of map because we care about order
type Pair struct {
	Key   string
	Value string
}

func NewPair(key string, value string) *Pair {
	p := new(Pair)
	p.Key = key
	p.Value = value
	return p
}

// Get value for a specific key
func getPairValue(p []Pair, key string) (string, error) {
	for _, v := range p {
		if v.Key == key {
			return v.Value, nil
		}
	}
	return "", errors.New("key not found")
}

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

func (c *Client) Connect(user string, password string) error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	// stash conn in instance
	c.conn = conn

	// try to log in
	res, err := c.Call("/login", nil)
	if err != nil {
		return err
	}
	
	// handle challenge/response
	challenge, err := getPairValue(res, "ret")
	if err != nil {
		return errors.New("Didn't get challenge from ROS")
	}
	h := md5.New()
	io.WriteString(h, "\x00")
	io.WriteString(h, password)
	io.WriteString(h, challenge)
	resp := fmt.Sprintf("%x", h.Sum(nil))
	var loginParams []Pair
	loginParams = append(loginParams, *NewPair("name", password))
	loginParams = append(loginParams, *NewPair("response", resp))
	
	res, err = c.Call("/login", loginParams)
	if err != nil {
		return err
	}
	fmt.Println(res)
	// handle challenge

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
