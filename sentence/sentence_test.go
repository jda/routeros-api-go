package sentence

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadWrite(t *testing.T) {
	for _, d := range []struct {
		sentence []string
	}{
		{[]string{"Hi"}},
		{strings.Split("a b c d e f", " ")},
	} {
		buf := &bytes.Buffer{}
		// Write sentence into buf.
		w := NewWriter(buf)
		for _, word := range d.sentence {
			w.WriteString(word)
		}
		w.WriteString("")
		// Read sentence from buf.
		r := NewReader(buf)
		s, _ := r.ReadSentence()
		if len(s) != len(d.sentence) {
			t.Fatalf("Expected sentence with %d words, got %d", len(d.sentence), len(s))
		}
		for i, word := range d.sentence {
			if word != string(s[i]) {
				t.Fatalf("Expected word %s at index %d, got %s", word, i, s[i])
			}
		}
	}
}
