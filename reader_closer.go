package textio

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// TokenReaderCloser extends TokenReader with explicit resource management.
//
// Implementations typically own one or more underlying resources
// (such as files or network connections) that must be released
// when Close is called.
type TokenReaderCloser interface {
	TokenReader
	io.Closer
}

// TokenStreamerCloser extends TokenStreamer with explicit resource management.
//
// Implementations must release all owned resources when Close is called.
type TokenStreamerCloser interface {
	TokenStreamer
	io.Closer
}

// TokenReaderStreamerCloser combines batch reading, streaming,
// and explicit resource management.
//
// This is the most complete contract and is typically implemented
// by types that own resources and support both access patterns.
type TokenReaderStreamerCloser interface {
	TokenReaderStreamer
	io.Closer
}

// [ReaderCloser] reads tokens from an io.Reader and optionally applies
// normalization and filtering before returning them.
//
// [ReaderCloser] supports both batch and streaming consumption patterns.
// The tokens read with [ReaderCloser] are either seperate with a string delimiter [delimiterStr] or a regular expression [delimiter].
// The readers that are closeable are stored and so can be close via [ReaderCloser] with Close function.
type ReaderCloser struct {
	*Reader
	closers []io.Closer
}

// NewReaderCloser creates a new ReaderCloser with default configuration.
//
// By default, the ReaderCloser reads from [os.Stdin], uses newline ("\n")
// as the token delimiter, applies the DefaultNormalizer, and
// fails on encountered errors.
//
// The returned ReaderCloser can be further configured using the
// provided setter methods before reading.
func NewReaderCloser() *ReaderCloser {
	r := NewReader()
	return &ReaderCloser{
		Reader: r,
	}
}

// [FromString] returns a shallow copy of the [ReaderCloser]
// with a new reader from string s. This discards and closes the previously set readers.
//
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) FromString(s string) *ReaderCloser {
	strReader := strings.NewReader(s)
	newR := *rc
	newR.SetReaders(strReader)
	return &newR
}

// [FromBytes] returns a shallow copy of the [ReaderCloser]
// with a new reader from the byte slice b. This discards and closes the previously set readers.
//
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) FromBytes(b []byte) *ReaderCloser {
	bytesReader := bytes.NewReader(b)
	newR := *rc
	newR.SetReaders(bytesReader)
	return &newR
}

// [FromFile] returns a shallow copy of the [ReaderCloser]
// with a new reader from the file. This discards and closes the previously set readers.
//
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) FromFile(path string) (*ReaderCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, newErrOpen(err)
	}
	newR := *rc
	newR.SetReaders(file)
	return &newR, nil
}

// WithDelimiter returns a shallow copy of the [ReaderCloser]
// configured with the given delimiter regular expression.
//
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) WithDelimiter(d *Delimiter) *ReaderCloser {
	newR := *rc
	newR.SetDelimiter(d)
	return &newR
}

// WithNormalizer returns a shallow copy of the [ReaderCloser]
// configured with the provided normalization function.
//
// The normalizer is applied to each token before filtering.
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) WithNormalizer(n NormalizeFunc) *ReaderCloser {
	newR := *rc
	newR.SetNormalizer(n)
	return &newR
}

// WithFilter returns a shallow copy of the [ReaderCloser]
// configured with the given filter function.
//
// The filter is evaluated after normalization.
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) WithFilter(f FilterFunc) *ReaderCloser {
	newR := *rc
	newR.SetFilter(f)
	return &newR
}

// WithReaders returns a shallow copy of the [ReaderCloser]
// configured with the given readers.
//
// The original [ReaderCloser] is not modified.
func (rc *ReaderCloser) WithReaders(readers ...io.Reader) *ReaderCloser {
	newR := *rc
	newR.SetReaders(readers...)
	return &newR
}

// [SetReaders] replaces the current input source with the provided readers.
//
// All readers are combined into a single stream using [io.MultiReader],
// and are consumed sequentially in the order they are provided.
//
// Any previously configured reader is discarded, and the closeable readers are closed.
func (rc *ReaderCloser) SetReaders(readers ...io.Reader) {
	_ = rc.Close()

	rc.closers = rc.closers[:0]

	var rs []io.Reader
	for _, r := range readers {
		rs = append(rs, r)
		if c, ok := r.(io.Closer); ok {
			rc.closers = append(rc.closers, c)
		}
	}

	rc.Reader.SetReaders(rs...)
}

// This discards the readers contained in [readers] field. The closeable readers are closed.
// If an error occures ([io.Closer] already closed) the function continues to close the others closeables. The first error that occured is wrapped in a [ErrClose] error and then is returned.
func (rc *ReaderCloser) Close() error {
	var firstErr error

	for _, c := range rc.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = newErrClose(err)
		}
	}

	rc.closers = nil
	return firstErr
}
