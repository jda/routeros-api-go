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
			return nil, fmt.Errorf("RouterOS device error: %s", reply.errorFrom(s))
		case "!re":
			reply.addSubPairs(s)
		default:
			return nil, fmt.Errorf("RouterOS device sent an unknown reply word: %s", s[0])
		}
	}
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

func (reply *Reply) errorFrom(sentence [][]byte) string {
	reply.addPairs(sentence)
	m, err := reply.GetPairVal("message")
	if err != nil || m == "" {
		return fmt.Sprintf("Unknown error: %v", sentence)
	}
	return m
}

func splitPairs(sentence [][]byte) [][]string {
	var pairs [][]string
	for _, word := range sentence {
		if bytes.Contains(word, []byte("=")) {
			t := bytes.SplitN(word, []byte("="), 3)
			if len(t) == 3 {
				pairs = append(pairs, []string{string(t[1]), string(t[2])})
			} else {
				pairs = append(pairs, []string{string(t[1]), ""})
			}
		}
	}
	return pairs
}
