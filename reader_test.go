package textio

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"
)

func stringReader(s string) io.Reader {
	return strings.NewReader(s)
}

func alphaOnlyFilter(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

func TestNewReader(t *testing.T) {
	r := NewReader()

	if r == nil {
		t.Fatal("NewReader() returned nil")
	}

	if r.reader == nil {
		t.Error("reader should not be nil")
	}

	if r.delimiter == nil {
		t.Error("delimiter should have default value")
	}

	if r.normalize == nil {
		t.Error("normalize should have default value")
	}

	if !r.FailOnError {
		t.Error("FailOnError should be true by default")
	}

	if r.FailOnInvalid {
		t.Error("FailOnInvalid should be false by default")
	}
}

func TestNewReaderWithDelimiter(t *testing.T) {
	regexp := regexp.MustCompile("\n")
	d := DefaultDelimiter()
	d.SetRegexp(regexp)
	r := NewReader()
	nr := r.WithDelimiter(d)

	if nr.delimiter.regexpr != regexp {
		t.Error("nr delimiter should have regexp value")
	}

	if r.delimiter.regexpr == regexp {
		t.Error("r delimiter should not have regexp value")
	}

}

func TestReadAll_Simple(t *testing.T) {
	input := "hello\nworld\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))

	tokens, err := r.ReadTokens()
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

func TestReadAll_EmptyInput(t *testing.T) {
	r := NewReader()
	r.SetReaders(stringReader(""))

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
}

func TestReadAll_WithNormalization(t *testing.T) {
	input := "  hello  \n  WORLD  \n  TeSt  "
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetNormalizer(ChainNormalizers(NormalizeUpper, NormalizeTrimSpace))

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"HELLO", "WORLD", "TEST"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestReadAll_WithFilter(t *testing.T) {
	input := "hello\nhi\nworld\na\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetFilter(FilterMinLength(3))

	tokens, err := r.ReadTokens()
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

func TestReadAll_WithFilterFailOnInvalid(t *testing.T) {
	input := "hello\nhi\nworld"
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetFilter(FilterMinLength(3))
	r.FailOnInvalid = true

	tokens, err := r.ReadTokens()
	if err == nil {
		t.Fatal("ReadTokens() should have returned an error")
	}

	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error should be ErrInvalid, got %T", err)
	}

	// Should have read "hello" before hitting "hi"
	if len(tokens) != 1 || tokens[0] != "hello" {
		t.Errorf("got tokens %v, want [hello]", tokens)
	}
}

func TestReadAll_EmptyLineBreak(t *testing.T) {
	input := "hello\nworld\ntest\n--end--"
	r := NewReader()
	r.SetReaders(stringReader(input))
	endDel := DefaultDelimiter()
	endDel.SetStr("--end--")
	r.SetEndDelimiter(endDel)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	// Should stop at empty line
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

func TestStream_Simple(t *testing.T) {
	input := "hello\nworld\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))

	ch := make(chan string, 10)
	errCh := make(chan error, 1)

	go func() {
		errCh <- r.StreamTokens(context.Background(), ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("StreamTokens() error = %v", err)
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

func TestStream_WithFilter(t *testing.T) {
	input := "hello\n123\nworld\n456\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetFilter(alphaOnlyFilter)

	ch := make(chan string, 10)
	errCh := make(chan error, 1)

	go func() {
		errCh <- r.StreamTokens(context.Background(), ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("StreamTokens() error = %v", err)
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

func TestStream_FailOnInvalid(t *testing.T) {
	input := "hello\n123\nworld"
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetFilter(alphaOnlyFilter)
	r.FailOnInvalid = true

	ch := make(chan string, 10)
	errCh := make(chan error, 1)

	go func() {
		errCh <- r.StreamTokens(context.Background(), ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	err := <-errCh
	if err == nil {
		t.Fatal("StreamTokens() should have returned an error")
	}

	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error should be ErrInvalid, got %T", err)
	}
}

func TestRead_Basic(t *testing.T) {
	input := "hello world"
	r := NewReader()
	r.SetReaders(stringReader(input))

	buf := make([]byte, 5)
	n, err := r.Read(buf)

	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if n != 5 {
		t.Errorf("Read() n = %d, want 5", n)
	}

	if string(buf) != "hello" {
		t.Errorf("Read() buf = %q, want %q", buf, "hello")
	}
}

func TestRead_EOF(t *testing.T) {
	input := "hi"
	r := NewReader()
	r.SetReaders(stringReader(input))

	buf := make([]byte, 10)
	n, err := r.Read(buf)

	if err != nil && err != io.EOF {
		t.Fatalf("Read() unexpected error = %v", err)
	}

	if n != 2 {
		t.Errorf("Read() n = %d, want 2", n)
	}

	if string(buf[:n]) != "hi" {
		t.Errorf("Read() buf = %q, want %q", buf[:n], "hi")
	}
}

func TestSetDelimiterStr_Comma(t *testing.T) {
	input := "one,two,three"
	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter()
	d.SetStr(",")
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"one", "two", "three"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestSetDelimiterStr_Semicolon(t *testing.T) {
	input := "apple;banana;cherry"
	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter()
	d.SetStr(";")
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"apple", "banana", "cherry"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestSetDelimiter_Regex(t *testing.T) {
	input := "one  two   three"
	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter()
	d.SetRegexp(regexp.MustCompile(`\s+`))
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"one", "two", "three"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestSetDelimiterFromString(t *testing.T) {
	input := "foo123bar456baz"
	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter()
	d.SetRegexp(regexp.MustCompile(`\d+`))
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"foo", "bar", "baz"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestSetDelimiterStr_Empty(t *testing.T) {
	input := "one\ntwo\nthree"
	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter()
	d.SetStr("")
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	// Should default to newline
	expected := []string{"one", "two", "three"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}
}

func TestSetReaders_Multiple(t *testing.T) {
	r1 := stringReader("hello\nworld\n")
	r2 := stringReader("foo\nbar\n")

	r := NewReader()
	r.SetReaders(r1, r2)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"hello", "world", "foo", "bar"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestAddReaders(t *testing.T) {
	r1 := stringReader("first\nsecond\n")
	r2 := stringReader("third\nfourth\n")

	r := NewReader()
	r.SetReaders(r1)
	r.AddReaders(r2)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"first", "second", "third", "fourth"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestRead_WithError(t *testing.T) {
	r := NewReader()
	r.SetReaders(errorReader{})

	buf := make([]byte, 10)
	_, err := r.Read(buf)

	if err == nil {
		t.Fatal("Read() should have returned an error")
	}

	if !errors.Is(err, ErrRead) {
		t.Errorf("error should be ErrRead, got %T", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("error should be also an io.ErrUnexpectedEOF")
	}
}

func TestReadAll_FailOnError(t *testing.T) {
	r := NewReader()
	r.SetReaders(errorReader{})
	r.FailOnError = true

	_, err := r.ReadTokens()

	if err == nil {
		t.Fatal("ReadTokens() should have returned an error")
	}

	if !errors.Is(err, ErrRead) {
		t.Errorf("error should be ErrRead, got %T", err)
	}
}

func TestDefaultNormalizer(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"\thello\t", "hello"},
		{"\nhello\n", "hello"},
		{"  hello world  ", "hello world"},
	}

	for _, tt := range tests {
		got := NormalizeTrimSpace(tt.input)
		if got != tt.want {
			t.Errorf("DefaultNormalizer(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReader_StreamTokens(t *testing.T) {
	input := "hello\nworld\nthis\nis\ngo"
	r := NewReader().FromString(input)
	out := make(chan string, 10) // buffer pour ne pas bloquer

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		if err := r.StreamTokens(ctx, out); err != nil {
			t.Errorf("StreamTokens returned error: %v", err)
		}
		close(out)
	}()

	expected := strings.Split(input, "\n")
	i := 0
	for token := range out {
		if i >= len(expected) {
			t.Errorf("Received more tokens than expected: %v", token)
			break
		}
		if token != expected[i] {
			t.Errorf("Token mismatch at index %d: got %q, want %q", i, token, expected[i])
		}
		i++
	}

	if i != len(expected) {
		t.Errorf("Number of tokens mismatch: got %d, want %d", i, len(expected))
	}
}

// Test cancellation
func TestReader_StreamTokens_Cancel(t *testing.T) {
	input := "a\nb\nc\nd\ne"
	r := NewReader().FromString(input)
	out := make(chan string) // petit buffer pour tester le cancel

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel() // cancel rapidement
	}()

	err := r.StreamTokens(ctx, out)
	if err == nil {
		t.Errorf("Expected context cancellation error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestIntegration_CompleteWorkflow(t *testing.T) {
	input := "  HELLO  \n  world  \n  123  \n  TeSt  \n  a  "

	r := NewReader().
		WithReaders(stringReader(input)).
		WithNormalizer(ChainNormalizers(NormalizeTrimSpace, NormalizeUpper)).
		WithFilter(FilterMinLength(3))

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"HELLO", "WORLD", "123", "TEST"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}

	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expected[i])
		}
	}
}

func TestIntegration_CSVParsing(t *testing.T) {
	input := "name,age,city\nAlice,30,NYC\nBob,25,LA"

	r := NewReader()
	r.SetReaders(stringReader(input))
	d := DefaultDelimiter().WithRegexpFromString(",|\n")
	r.SetDelimiter(d)

	tokens, err := r.ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens() error = %v", err)
	}

	expected := []string{"name", "age", "city", "Alice", "30", "NYC", "Bob", "25", "LA"}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}
}

func BenchmarkReadAll_Small(b *testing.B) {
	input := "hello\nworld\ntest\nfoo\nbar"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader()
		r.SetReaders(stringReader(input))
		_, _ = r.ReadTokens()
	}
}

func BenchmarkReadAll_Large(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("word")
		sb.WriteString("\n")
	}
	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader()
		r.SetReaders(stringReader(input))
		_, _ = r.ReadTokens()
	}
}

func BenchmarkStream_Large(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("word")
		sb.WriteString("\n")
	}
	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader()
		r.SetReaders(stringReader(input))
		ch := make(chan string, 100)

		go func() {
			_ = r.StreamTokens(context.Background(), ch)
			close(ch)
		}()

		for range ch {
		}
	}
}
