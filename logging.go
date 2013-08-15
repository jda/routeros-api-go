package routeros

import (
	"github.com/jcelliott/lumber"
)

// Enabled logging by providing a Logger instance from Lumber.
func (c *Client) Logging(l *lumber.Logger, level string) {
	c.logger = l
	if level == "DEBUG" {
		c.debug = true
	}
}