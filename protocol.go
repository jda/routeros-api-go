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
}

func NewRouterOSClient(address string) *Client {

}

func (c *Client) Connect(user string, password string) {

}
