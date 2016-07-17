// Package routeros provides a programmatic interface to the Mikrotik RouterOS API
package routeros

import (
	"fmt"
	"io"
	"strings"
)

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

// Get reply
func (c *Client) receive() (reply Reply, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		e, ok := r.(mtbyteprotoError)
		if ok {
			err = e
			return
		}
		panic(r)
	}()

	re := false
	done := false
	trap := false
	subReply := make(map[string]string, 1)
	for {
		length := c.getlen()
		if length == 0 && done {
			break
		}

		inbuf := make([]byte, length)
		n, err := io.ReadAtLeast(c.conn, inbuf, int(length))
		// We don't actually care about EOF, but things like ErrUnspectedEOF we would
		if err != nil && err != io.EOF {
			return reply, err
		}

		// be annoying about reading exactly the correct number of bytes
		if int64(n) != length {
			return reply, fmt.Errorf("incorrect number of bytes read")
		}

		word := string(inbuf)
		if word == "!done" {
			done = true
			continue
		}

		if word == "!trap" { // error reply
			trap = true
			continue
		}

		if word == "!re" { // new term so start a new pair
			if len(subReply) > 0 {
				// we've already used this subreply because it has stuff in it
				// so we need to close it out and make a new one
				reply.SubPairs = append(reply.SubPairs, subReply)
				subReply = make(map[string]string, 1)
			} else {
				re = true
			}
			continue
		}

		if strings.Contains(word, "=") {
			parts := strings.SplitN(word, "=", 3)
			var key, val string
			if len(parts) == 3 {
				key = parts[1]
				val = parts[2]
			} else {
				key = parts[1]
			}

			if re {
				if key != "" {
					subReply[key] = val
				}
			} else {
				var p Pair
				p.Key = key
				p.Value = val
				reply.Pairs = append(reply.Pairs, p)
			}
		}
	}

	if len(subReply) > 0 {
		reply.SubPairs = append(reply.SubPairs, subReply)
	}

	// if we got a error flag from routeros, look for a message and signal err
	if trap {
		trapMesasge := ""
		for _, v := range reply.Pairs {
			if v.Key == "message" {
				trapMesasge = v.Value
				continue
			}
		}

		if trapMesasge == "" {
			return reply, fmt.Errorf("routeros: unknown error")
		} else {
			return reply, fmt.Errorf("routeros: %s", trapMesasge)
		}
	}

	return reply, nil
}
