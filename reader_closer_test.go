package textio

import (
	"testing"
)

func TestClose(t *testing.T) {
	rc, err := NewReaderCloser().FromFile("reader_closer_test.txt")
	if err != nil {
		t.Fatalf("Error opening test file: %v", err)
	}
	defer rc.Close()

	endDelim := DefaultDelimiter().WithStr("--stop--")
	rc.SetEndDelimiter(endDelim)

	tokens, err := rc.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"hello", "world", "test"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}
