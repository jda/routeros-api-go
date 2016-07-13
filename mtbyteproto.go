package routeros

import (
	"bytes"
)

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
