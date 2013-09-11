// Package routeros provides a programmatic interface to to the Mikrotik RouterOS API
package routeros

import (
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
func (c *Client) receive() (Reply, error) {
	var reply Reply

	re := false
	done := false
	subReply := make(map[string]string, 1)
	for {
		length := c.getlen()
		if length == 0 && done {
			break
		}

		inbuf := make([]byte, length)
		c.conn.Read(inbuf)
		word := string(inbuf)

		if word == "!done" {
			done = true
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
			key := parts[1]
			val := parts[2]

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

	return reply, nil
}
