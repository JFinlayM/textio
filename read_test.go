package textio

import (
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
)

func stringReader(s string) io.Reader {
	return strings.NewReader(s)
}

func toUpperNormalizer(s string, ctx any) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func minLengthFilter(minLen int) FilterFunc {
	return func(s string, ctx any) bool {
		return len(s) >= minLen
	}
}

func alphaOnlyFilter(s string, ctx any) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

type testContext struct {
	counter int
}

func countingNormalizer(s string, ctx any) string {
	if tc, ok := ctx.(*testContext); ok {
		tc.counter++
	}
	return strings.TrimSpace(s)
}

func TestNewReader(t *testing.T) {
	r := NewReader()

	if r == nil {
		t.Fatal("NewReader() returned nil")
	}

	if r.reader == nil {
		t.Error("reader should not be nil")
	}

	if r.delimiterStr == "" {
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

func TestReadAll_Simple(t *testing.T) {
	input := "hello\nworld\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
}

func TestReadAll_WithNormalization(t *testing.T) {
	input := "  hello  \n  WORLD  \n  TeSt  "
	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetNormalizer(toUpperNormalizer)

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetFilter(minLengthFilter(3))

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetFilter(minLengthFilter(3))
	r.FailOnInvalid = true

	tokens, err := r.ReadAll()
	if err == nil {
		t.Fatal("ReadAll() should have returned an error")
	}

	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error should be ErrInvalid, got %T", err)
	}

	// Should have read "hello" before hitting "hi"
	if len(tokens) != 1 || tokens[0] != "hello" {
		t.Errorf("got tokens %v, want [hello]", tokens)
	}
}

func TestReadAll_WithUserContext(t *testing.T) {
	input := "one\ntwo\nthree"
	ctx := &testContext{}

	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetNormalizer(countingNormalizer)
	r.UserContext = ctx

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(tokens) != 3 {
		t.Errorf("got %d tokens, want 3", len(tokens))
	}

	if ctx.counter != 3 {
		t.Errorf("normalizer called %d times, want 3", ctx.counter)
	}
}

func TestReadAll_EmptyLineBreak(t *testing.T) {
	input := "hello\nworld\n\ntest"
	r := NewReader()
	r.SetReaders(stringReader(input))

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Should stop at empty line
	expected := []string{"hello", "world"}
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
		errCh <- r.Stream(ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream() error = %v", err)
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
		errCh <- r.Stream(ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream() error = %v", err)
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
		errCh <- r.Stream(ch)
		close(ch)
	}()

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	err := <-errCh
	if err == nil {
		t.Fatal("Stream() should have returned an error")
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
	r.SetDelimiterStr(",")

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetDelimiterStr(";")

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetDelimiter(regexp.MustCompile(`\s+`))

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetDelimiterFromString(`\d+`)

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetDelimiterStr("")

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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

	_, err := r.ReadAll()

	if err == nil {
		t.Fatal("ReadAll() should have returned an error")
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
		got := DefaultNormalizer(tt.input, nil)
		if got != tt.want {
			t.Errorf("DefaultNormalizer(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIntegration_CompleteWorkflow(t *testing.T) {
	input := "  HELLO  \n  world  \n  123  \n  TeSt  \n  a  "

	r := NewReader()
	r.SetReaders(stringReader(input))
	r.SetNormalizer(toUpperNormalizer)
	r.SetFilter(minLengthFilter(3))

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
	r.SetDelimiterFromString(",|\n")

	tokens, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
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
		_, _ = r.ReadAll()
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
		_, _ = r.ReadAll()
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
			_ = r.Stream(ch)
			close(ch)
		}()

		for range ch {
		}
	}
}
