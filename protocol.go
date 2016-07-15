// Package routeros provides a programmatic interface to the Mikrotik RouterOS API
package routeros

import (
	"fmt"

	"joi.com.br/mikrotik-go/sentence"
)

func (r *Reply) addSentence(sen *sentence.Sentence) (bool, error) {
	switch sen.Word {
	case "!done":
		r.Done = sen
		return true, nil
	case "!trap", "!fatal":
		return true, &DeviceError{sen}
	case "!re":
		r.Re = append(r.Re, sen)
	case "":
		// API docs say that empty sentences should be ignored
	default:
		return true, &UnknownReplyError{sen}
	}
	return false, nil
}

// readReply reads one reply synchronously. It returns the reply.
func (c *Client) readReply() (*Reply, error) {
	reply := &Reply{}
	for {
		sen, err := c.sentenceReader.ReadSentence()
		if err != nil {
			return nil, err
		}
		done, err := reply.addSentence(sen)
		if err != nil {
			return nil, err
		}
		if done {
			return reply, nil
		}
	}
}

// Loop starts asynchronous mode. It only returns if there is an error.
func (c *Client) Loop() error {
	c.async = true
	defer func() { c.async = false }()

	c.tags = make(map[string]*AsyncReply)
	defer func() {
		for _, a := range c.tags {
			close(a.C)
		}
	}()

	for {
		sen, err := c.sentenceReader.ReadSentence()
		if err != nil {
			return err
		}
		c.Lock()
		a, ok := c.tags[sen.Tag]
		c.Unlock()
		if !ok {
			continue
		}
		done, err := a.Reply.addSentence(sen)
		if err != nil {
			a.Err = err
		}
		if done {
			close(a.C)
		}
	}
}

// UnknownReplyError records the sentence whose Word is unknown.
type UnknownReplyError struct {
	Sentence *sentence.Sentence
}

func (err *UnknownReplyError) Error() string {
	return fmt.Sprintf("unknown RouterOS reply word: %s", err.Sentence.Word)
}

// DeviceError records the sentence containing the error received from the device.
// The sentence may have Word !trap or !fatal.
type DeviceError struct {
	Sentence *sentence.Sentence
}

func (err *DeviceError) Error() string {
	m := err.Sentence.Map["message"]
	if m == "" {
		m = fmt.Sprintf("unknown: %s", err.Sentence)
	}
	return fmt.Sprintf("RouterOS: %s", m)
}
