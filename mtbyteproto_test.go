package routeros

import (
	"bytes"
	"net"
	"testing"
	"time"
)

const (
	mtCodingValue1 = 0x00000001
	mtCodingValue2 = 0x00000087
	mtCodingValue3 = 0x00004321
	mtCodingValue4 = 0x002acdef
	mtCodingValue5 = 0x10000080
)

var (
	mtCodingSize1 = []byte{0x01}
	mtCodingSize2 = []byte{0x80, 0x87}
	mtCodingSize3 = []byte{0xC0, 0x43, 0x21}
	mtCodingSize4 = []byte{0xE0, 0x2a, 0xcd, 0xef}
	mtCodingSize5 = []byte{0xF0, 0x10, 0x00, 0x00, 0x80}
)

// Create a test net.Conn type
type testConn struct {
	readBuf bytes.Buffer
}

func (testConn) LocalAddr() net.Addr                { return nil }
func (testConn) RemoteAddr() net.Addr               { return nil }
func (testConn) SetDeadline(t time.Time) error      { return nil }
func (testConn) SetReadDeadline(t time.Time) error  { return nil }
func (testConn) SetWriteDeadline(t time.Time) error { return nil }
func (c *testConn) Read(b []byte) (int, error)      { return c.readBuf.Read(b) }
func (testConn) Write(b []byte) (int, error)        { return 0, nil }
func (testConn) Close() error                       { return nil }

// Test Client.getlen decodings
func TestGetlen(t *testing.T) {
	c := &testConn{readBuf: *bytes.NewBuffer(mtCodingSize1)}
	mc := &Client{conn: c}
	len := mc.getlen()
	if len != mtCodingValue1 || c.readBuf.Len() != 0 {
		t.Errorf("single byte read failed, got %#08x", len)
	}
	c.readBuf = *bytes.NewBuffer(mtCodingSize2)
	len = mc.getlen()
	if len != mtCodingValue2 || c.readBuf.Len() != 0 {
		t.Errorf("double byte read failed, got %#08x", len)
	}
	c.readBuf = *bytes.NewBuffer(mtCodingSize3)
	len = mc.getlen()
	if len != mtCodingValue3 || c.readBuf.Len() != 0 {
		t.Errorf("triple byte read failed, got %#08x", len)
	}
	c.readBuf = *bytes.NewBuffer(mtCodingSize4)
	len = mc.getlen()
	if len != mtCodingValue4 || c.readBuf.Len() != 0 {
		t.Errorf("quad byte read failed, got %#08x", len)
	}
	c.readBuf = *bytes.NewBuffer(mtCodingSize5)
	len = mc.getlen()
	if len != mtCodingValue5 || c.readBuf.Len() != 0 {
		t.Errorf("penta byte read failed, got %#08x", len)
	}
}

// Test prefixlen encodings
func TestPrefixLen(t *testing.T) {
	b := prefixlen(mtCodingValue1).Bytes()
	if !bytes.Equal(mtCodingSize1, b) {
		t.Errorf("single byte write failed, got %v", b)
	}
	b = prefixlen(mtCodingValue2).Bytes()
	if !bytes.Equal(mtCodingSize2, b) {
		t.Errorf("double byte write failed, got %v", b)
	}
	b = prefixlen(mtCodingValue3).Bytes()
	if !bytes.Equal(mtCodingSize3, b) {
		t.Errorf("triple byte write failed, got %v", b)
	}
	b = prefixlen(mtCodingValue4).Bytes()
	if !bytes.Equal(mtCodingSize4, b) {
		t.Errorf("quad byte write failed, got %v", b)
	}
	b = prefixlen(mtCodingValue5).Bytes()
	if !bytes.Equal(mtCodingSize5, b) {
		t.Errorf("penta byte write failed, got %v", b)
	}
}
