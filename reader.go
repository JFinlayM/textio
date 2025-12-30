// Package [textio] provides configurable and extensible text input utilities.
//
// The package is designed to read text data from one or multiple [io.Reader]
// sources, tokenize it using a configurable delimiter, and optionally
// normalize, filter, and validate the extracted text.
//
// [textio] focuses on flexibility rather than strict formats. It allows callers
// to plug custom normalization logic, filtering rules, and error formatting,
// making it suitable for tasks such as text parsing, word extraction,
// preprocessing pipelines, or command-line input processing.
//
// The core abstraction is the Reader type, which wraps one or more [io.Reader]
// instances and exposes a controlled and configurable reading behavior.
package textio

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"strings"
)

// [Reader] reads tokens from an io.Reader and optionally applies
// normalization and filtering before returning them.
//
// [Reader] supports both batch and streaming consumption patterns.
// The tokens read with [Reader] are either seperate with a string delimiter [delimiterStr] or a regular expression [delimiter]
type Reader struct {
	// The reader(s) from where we read tokens
	reader io.Reader
	// delimiter is for the seperation of the tokens.
	delimiter *Delimiter
	// endDelimiter is used to stop scanning. Keep in mind that in order to stop scanning the input NEEDS the [delimiter] followed by [endDelimiter].
	// For example: input = "hello\nworld\nend", returns ["hello", "world"] if delimiter = "\n" and endDelimiter = "end".
	endDelimiter  *Delimiter
	normalize     NormalizeFunc
	filter        FilterFunc
	FailOnError   bool
	FailOnInvalid bool
}

// NewReader creates a new Reader with default configuration.
//
// By default, the Reader reads from [os.Stdin], uses newline ("\n")
// as the token delimiter, applies the DefaultNormalizer, and
// fails on encountered errors.
//
// The returned Reader can be further configured using the
// provided setter methods before reading.
func NewReader() *Reader {
	return &Reader{
		reader:      os.Stdin,
		delimiter:   DefaultDelimiter(),
		normalize:   NormalizeTrimSpace,
		FailOnError: true,
	}
}

// [FromString] returns a shallow copy of the [Reader]
// with a new reader from string s.
//
// The original [Reader] is not modified.
func (r *Reader) FromString(s string) *Reader {
	strReader := strings.NewReader(s)
	newR := *r
	newR.SetReaders(strReader)
	return &newR
}

// [FromBytes] returns a shallow copy of the [Reader]
// with a new reader from the byte slice b.
//
// The original [Reader] is not modified.
func (r *Reader) FromBytes(b []byte) *Reader {
	bytesReader := bytes.NewReader(b)
	newR := *r
	newR.SetReaders(bytesReader)
	return &newR
}

// WithDelimiter returns a shallow copy of the [Reader]
// configured with the given delimiter regular expression.
//
// The original [Reader] is not modified.
func (r *Reader) WithDelimiter(d *Delimiter) *Reader {
	newR := *r
	newR.SetDelimiter(d)
	return &newR
}

// WithNormalizer returns a shallow copy of the [Reader]
// configured with the provided normalization function.
//
// The normalizer is applied to each token before filtering.
// The original [Reader] is not modified.
func (r *Reader) WithNormalizer(n NormalizeFunc) *Reader {
	newR := *r
	newR.SetNormalizer(n)
	return &newR
}

// WithFilter returns a shallow copy of the [Reader]
// configured with the given filter function.
//
// The filter is evaluated after normalization.
// The original [Reader] is not modified.
func (r *Reader) WithFilter(f FilterFunc) *Reader {
	newR := *r
	newR.SetFilter(f)
	return &newR
}

// WithReaders returns a shallow copy of the [Reader]
// configured with the given readers.
//
// The original [Reader] is not modified.
func (r *Reader) WithReaders(readers ...io.Reader) *Reader {
	newR := *r
	newR.SetReaders(readers...)
	return &newR
}

// [SetReaders] replaces the current input source with the provided readers.
//
// All readers are combined into a single stream using [io.MultiReader],
// and are consumed sequentially in the order they are provided.
//
// Any previously configured reader is discarded.
func (r *Reader) SetReaders(readers ...io.Reader) {
	r.reader = io.MultiReader(readers...)
}

// [AddReaders] appends the provided readers to the existing input source.
//
// The existing reader is preserved and the new readers are appended
// after it, forming a single sequential stream via [io.MultiReader].
//
// This allows additional input sources to be added without
// replacing the current reader.
func (r *Reader) AddReaders(readers ...io.Reader) {
	readers = append([]io.Reader{r.reader}, readers...)
	r.SetReaders(readers...)
}

// Sets the delimiter used to seperate input into tokens.
// This resets the [delimiterStr] field of r.
func (r *Reader) SetDelimiter(d *Delimiter) {
	r.delimiter = d
}

// Sets the delimiter used to seperate input into tokens.
// This resets the [delimiterStr] field of r.
func (r *Reader) SetEndDelimiter(d *Delimiter) {
	r.endDelimiter = d
}

// Sets the function to be called to normalize current read token before passing through filter function. There is none by default.
func (r *Reader) SetNormalizer(normalizeFunc NormalizeFunc) {
	r.normalize = normalizeFunc
}

// Sets the function to be called to filter current read token. Should return true is the token satisfies user defined constraints, false otherwise.
func (r *Reader) SetFilter(filterFunc FilterFunc) {
	r.filter = filterFunc
}

// Read processes input from the provided [io.Reader](s).
// It reads strings, applies normalization and filtering if specified,
// and returns the resulting strings or an error if any issues occur.
// If a end delimiter is set, then the scanning stop if there is a token matching seperation delimiter followed by end delimiter.
// For example: input = "hello\nworld\nend", returns ["hello", "world"] if delimiter = "\n" and endDelimiter = "end".
//
// Returns:
//   - A slice of strings containing the processed input.
//   - error: [ErrInvalid] if the token doesnt respect constraints defined by filter function and if [FailOnInvalid] is set. [ErrRead] if an error occured during scanning.
//
// Behavior:
//   - If a delimiter is specified in the [Reader], it uses a custom split function
//     to tokenize the input; otherwise, it defaults to line-based scanning.
//   - If a normalization function is provided, it applies the function to each string read.
//   - If a filtering function is provided, it validates each string against the filter.
//     If a string fails the filter and FailOnInvalid is true, the function returns an error. Otherwise, it skips the invalid string.
//   - If an error occurs during scanning and FailOnError is true, the function returns the error.
func (r *Reader) ReadTokens() ([]string, error) {
	var tokens []string

	scanner := bufio.NewScanner(r.reader)
	scanner.Split(r.delimiter.SplitFunc())

	n := 0
	for scanner.Scan() {
		token := scanner.Text()
		if r.endDelimiter != nil && r.endDelimiter.MatchString(token) {
			break
		}

		if r.normalize != nil {
			token = r.normalize(token)
		}

		if r.filter != nil && !r.filter(token) {
			if r.FailOnInvalid {
				return tokens, newErrInvalid(token, n)
			}
			n += len(token)
			continue
		}

		n += len(token)
		tokens = append(tokens, token)
	}

	if err := scanner.Err(); err != nil && r.FailOnError {
		return tokens, newErrRead(err)
	}

	return tokens, nil
}

// Read processes input from the provided [io.Reader](s).
// It populates 0 <= n <= len(p) bytes from the files in p,
// and returns an error if any issues occur.
//
// Returns:
//   - n: number of bytes read
//   - err: [ErrRead] if any issues occur during reading
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if err != nil {
		err = newErrRead(err)
	}
	return n, err
}

// StreamTokens reads tokens from the Reader's input source and sends them to the provided channel.
//
// Tokens are extracted according to the Reader's configured delimiter (string or regex),
// normalized using the optional normalization function, and filtered using the optional filter function.
// If a end delimiter is set, then the scanning stop if there is a token matching seperation delimiter followed by end delimiter.
// For example: input = "hello\nworld\nend", returns ["hello", "world"] if delimiter = "\n" and endDelimiter = "end".
//
// The function respects context cancellation via the provided `ctx`. If `ctx` is canceled,
// StreamTokens returns immediately with `ctx.Err()`.
//
// Parameters:
//   - ctx: context to control cancellation and deadlines.
//   - out: channel to which valid tokens are sent.
//
// Returns:
//   - error:
//   - ErrInvalid if a token fails the filter and FailOnInvalid is true.
//   - ErrRead if a scanning or I/O error occurs and FailOnError is true.
//   - ctx.Err() if the context is canceled.
//
// Behavior:
//   - Tokens are read sequentially from the Reader's input sources.
//   - Normalization is applied before filtering.
//   - Tokens that fail the filter are skipped unless FailOnInvalid is set.
//   - The function terminates when all input is consumed, an error occurs, or the context is canceled.
func (r *Reader) StreamTokens(ctx context.Context, out chan string) error {
	scanner := bufio.NewScanner(r.reader)
	scanner.Split(r.delimiter.SplitFunc())

	n := 0
	for scanner.Scan() {
		token := scanner.Text()

		if r.endDelimiter != nil && r.endDelimiter.MatchString(token) {
			return nil
		}

		if r.normalize != nil {
			token = r.normalize(token)
		}

		if r.filter != nil && !r.filter(token) {
			if r.FailOnInvalid {
				return newErrInvalid(token, n)
			}
			n += len(token)
			continue
		}

		n += len(token)
		select {
		case out <- token:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil && r.FailOnError {
		return newErrRead(err)
	}
	return nil
}
