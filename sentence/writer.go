package sentence

import "io"

// Writer writes words to a RouterOS device.
type Writer interface {
	WriteString(word string)
	Err() error
}

type writer struct {
	io.Writer
	err error
}

// NewWriter returns a new Writer to write to w.
func NewWriter(w io.Writer) Writer {
	return &writer{Writer: w}
}

// Err returns the last error that occurred on this Writer.
func (w *writer) Err() error {
	return w.err
}

// WriteString writes one RouterOS word.
func (w *writer) WriteString(word string) {
	w.writeBytes([]byte(word))
}

func (w *writer) writeBytes(word []byte) {
	if w.err != nil {
		return
	}
	err := w.writeLength(len(word))
	if err != nil {
		w.err = err
		return
	}
	_, err = w.Write(word)
	if err != nil {
		w.err = err
		return
	}
}

func (w *writer) writeLength(l int) error {
	_, err := w.Write(encodeLength(l))
	return err
}

func encodeLength(l int) []byte {
	switch {
	case l < 0x80:
		return []byte{byte(l)}
	case l < 0x4000:
		return []byte{byte(l>>8) | 0x80, byte(l)}
	case l < 0x200000:
		return []byte{byte(l>>16) | 0xC0, byte(l >> 8), byte(l)}
	case l < 0x10000000:
		return []byte{byte(l>>24) | 0xE0, byte(l >> 16), byte(l >> 8), byte(l)}
	default:
		return []byte{0xF0, byte(l >> 24), byte(l >> 16), byte(l >> 8), byte(l)}
	}
}
