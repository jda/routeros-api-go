// Package routeros provides a programmatic interface to the Mikrotik RouterOS API
package routeros

import (
	"fmt"

	"joi.com.br/mikrotik-go/sentence"
)

// Get reply
func (c *Client) readReply() (*Reply, error) {
	reply := &Reply{}
	for {
		s, err := c.sentenceReader.ReadSentence()
		if err != nil {
			return nil, err
		}
		switch s.Word {
		case "!done":
			reply.Done = s
			return reply, nil
		case "!trap", "!fatal":
			return nil, &DeviceError{s}
		case "!re":
			reply.Re = append(reply.Re, s)
		case "":
			// API docs say that empty sentences should be ignored
		default:
			return nil, &UnknownReplyError{s}
		}
	}
}

type UnknownReplyError struct {
	Sentence *sentence.Sentence
}

func (err *UnknownReplyError) Error() string {
	return fmt.Sprintf("unknown RouterOS reply word: %s", err.Sentence.Word)
}

type DeviceError struct {
	Trap *sentence.Sentence
}

func (err *DeviceError) Error() string {
	m := err.Trap.Map["message"]
	if m == "" {
		m = fmt.Sprintf("unknown: %s", err.Trap)
	}
	return fmt.Sprintf("RouterOS: %s", m)
}
