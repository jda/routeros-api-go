// Package routeros provides a Client interface to to the Mikrotik RouterOS API
package routeros

import (
	"github.com/jcelliott/lumber"
	"net"
)

// A Client is a RouterOS API client.
type Client struct {
	// Network Address.
	// E.g. "10.0.0.1:8728" or "router.example.com:8728"
	address  string
	user     string
	password string
	logger   *lumber.Logger
	// debug logging enabled
	debug	 bool
	// Ready for work (login ok and connection not terminated)
	ready 	 bool
}

type Query struct {
	Word string
}

func NewRouterOSClient(address string) *Client {

}

func (c *Client) Connect(user string, password string) error {
	return nil
}

// Enabled logging by providing a Logger instance from Lumber. 
func (c *Client) Logging(l *lumber.Logger, level string) {
	c.logger = l
	if level == "DEBUG" {
		c.debug = true
	}
}


