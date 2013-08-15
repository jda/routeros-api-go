package routeros

import (
	"bytes"
	"strconv"
)

// Get just one byte because MT's size prefix is overoptimized
func (c *Client) getone() int {
	charlet := make([]byte, 1)
	c.conn.Read(charlet)
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
// TODO: based on MT Docs. Look for way to make this cleaner later
func prefixlen(l int) bytes.Buffer {
	var b bytes.Buffer

	if l < 0x80 {
		b.Write([]byte(string(l)))
	} else if l < 0x4000 {
		l |= 0x8000
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else if l < 0x200000 {
		l |= 0xC00000
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else if l < 0x10000000 {
		l |= 0xE0000000
		b.Write([]byte(strconv.Itoa((l >> 24) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	} else {
		b.Write([]byte(strconv.Itoa(0xF0)))
		b.Write([]byte(strconv.Itoa((l >> 24) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 16) & 0xFF)))
		b.Write([]byte(strconv.Itoa((l >> 8) & 0xFF)))
		b.Write([]byte(strconv.Itoa(l & 0xFF)))
	}

	return b
}
