package routeros

import (
	"bytes"
)

type mtbyteprotoError error

// Get just one byte because MT's size prefix is overoptimized
func (c *Client) getone() int {
	charlet := make([]byte, 1)
	_, err := c.conn.Read(charlet)
	if err != nil {
		panic(mtbyteprotoError(err))
	}
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
func prefixlen(l int) *bytes.Buffer {
	var b bytes.Buffer
	switch {
	case l < 0x80:
		b.WriteByte(byte(l))
	case l < 0x4000:
		b.WriteByte(byte(l>>8) | 0x80)
		b.WriteByte(byte(l))
	case l < 0x200000:
		b.WriteByte(byte(l>>16) | 0xC0)
		b.WriteByte(byte(l >> 8))
		b.WriteByte(byte(l))
	case l < 0x10000000:
		b.WriteByte(byte(l>>24) | 0xE0)
		b.WriteByte(byte(l >> 16))
		b.WriteByte(byte(l >> 8))
		b.WriteByte(byte(l))
	default:
		b.WriteByte(0xF0)
		b.WriteByte(byte(l >> 24))
		b.WriteByte(byte(l >> 16))
		b.WriteByte(byte(l >> 8))
		b.WriteByte(byte(l))
	}
	return &b
}
