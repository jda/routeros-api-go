// Package routeros provides a programmatic interface to the Mikrotik RouterOS API
package routeros

import (
	"bytes"
	"fmt"
)

// Get reply
func (c *Client) receive() (*Reply, error) {
	reply := &Reply{}
	for {
		s, err := c.sentenceReader.ReadSentence()
		if err != nil {
			return nil, err
		}
		if len(s) == 0 {
			continue // API docs say that empty sentences should be ignored
		}
		switch string(s[0]) {
		case "!done":
			reply.addPairs(s[1:])
			return reply, nil
		case "!trap", "!fatal":
			return nil, reply.errorFrom(s)
		case "!re":
			reply.addSubPairs(s[1:])
		default:
			return nil, &UnknownReplyError{s[0]}
		}
	}
}

type UnknownReplyError struct {
	Word []byte
}

func (err *UnknownReplyError) Error() string {
	return fmt.Sprintf("unknown RouterOS reply word: %s", err.Word)
}

func (reply *Reply) addPairs(sentence [][]byte) {
	for _, pair := range splitPairs(sentence) {
		reply.Pairs = append(reply.Pairs, Pair{
			Key:   pair[0],
			Value: pair[1],
		})
	}
}

func (reply *Reply) addSubPairs(sentence [][]byte) {
	pairs := make(map[string]string)
	for _, pair := range splitPairs(sentence) {
		pairs[pair[0]] = pair[1]
	}
	reply.SubPairs = append(reply.SubPairs, pairs)
}

type DeviceError struct {
	Message string
}

func (err *DeviceError) Error() string {
	return fmt.Sprintf("RouterOS: %s", err.Message)
}

func (reply *Reply) errorFrom(sentence [][]byte) error {
	reply.addPairs(sentence)
	m, err := reply.GetPairVal("message")
	if err != nil || m == "" {
		m = fmt.Sprintf("unknown: %q", sentence)
	}
	return &DeviceError{m}
}

func splitPairs(sentence [][]byte) [][]string {
	var pairs [][]string
	for _, word := range sentence {
		if bytes.HasPrefix(word, []byte("=")) {
			t := bytes.SplitN(word[1:], []byte("="), 2)
			if len(t) == 2 {
				pairs = append(pairs, []string{string(t[0]), string(t[1])})
			} else {
				pairs = append(pairs, []string{string(t[0]), ""})
			}
		}
	}
	return pairs
}
