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
	"io"
	"os"
	"regexp"
	"strings"
)

// s is the string currently being read, ctx parameter is set as the [UserContext] attribute
// Used to transform token before passing through the [FilterFunc].
type NormalizeFunc func(s string, ctx any) string

// s is the string currently being read, ctx parameter is set as the [UserContext] attribute.
// Should return true is the token satisfies user defined constraints, false otherwise.
type FilterFunc func(s string, ctx any) bool

// [Reader] reads tokens from an io.Reader and optionally applies
// normalization and filtering before returning them.
//
// [Reader] supports both batch and streaming consumption patterns.
// The tokens read with [Reader] are either seperate with a string delimiter [delimiterStr] or a regular expression [delimiter]
type Reader struct {
	reader        io.Reader
	delimiter     *regexp.Regexp
	delimiterStr  string
	normalize     NormalizeFunc
	filter        FilterFunc
	UserContext   any
	FailOnError   bool
	FailOnInvalid bool
}

// Default normalization function. It is a wrapper for the [strings.TrimSpace] function from the strings module.
func DefaultNormalizer(s string, ctx any) string {
	return strings.TrimSpace(s)
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
		reader:       os.Stdin,
		delimiterStr: "\n",
		normalize:    DefaultNormalizer,
		FailOnError:  true,
	}
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
	r.reader = io.MultiReader(readers...)
}

// Sets the [delimiterStr] field of r used to seperate input into tokens.
// This resets the [delimiter] field of r.
func (r *Reader) SetDelimiterStr(s string) {
	if s == "" {
		s = "\n"
	}
	r.delimiterStr = s
	r.delimiter = nil
}

// Sets the delimiter used to seperate input into tokens.
// This resets the [delimiterStr] field of r.
func (r *Reader) SetDelimiter(regexpr *regexp.Regexp) {
	r.delimiter = regexpr
	r.delimiterStr = ""
}

// Sets the delimiter used to seperate input into tokens.
// This resets the [delimiterStr] field of r.
// This function will panic if the expression cannot compile.
func (r *Reader) SetDelimiterFromString(expr string) {
	regexpr := regexp.MustCompile(expr)
	r.delimiter = regexpr
	r.delimiterStr = ""
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
func (r *Reader) ReadAll() ([]string, error) {
	var tokens []string

	scanner := bufio.NewScanner(r.reader)
	scanner.Split(r.createSplitFunc())

	n := 0
	for scanner.Scan() {
		token := scanner.Text()
		if token == "" {
			break
		}
		if r.normalize != nil {
			token = r.normalize(token, r.UserContext)
		}

		if r.filter != nil && !r.filter(token, r.UserContext) {
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

// Read processes input from the provided [io.Reader](s).
// It reads strings, applies normalization and filtering if specified.
// The resulting strings are passed through the `s` channel or an error is returned if any issues occur.
//
// Parameters:
//   - s: the string channel
//
// Returns:
//   - error: [ErrInvalid] if the token doesnt respect constraints defined by filter function and if [FailOnInvalid] is set. [ErrRead] if an error occured during scanning.
//
// Behavior:
//   - If a delimiter is specified in the [Reader], it uses a custom split function
//     to tokenize the input; otherwise, it defaults to line-based scanning.
//   - If a normalization function is provided, it applies the function to each string read.
//   - If a filtering function is provided, it validates each string against the filter.
//     If a string fails the filter and FailOnInvalid is true, the function returns an error. Otherwise, it skips the invalid string.
//   - If an error occurs during scanning and FailOnError is true, the function returns the error.
func (r *Reader) Stream(s chan string) error {
	scanner := bufio.NewScanner(r.reader)
	scanner.Split(r.createSplitFunc())

	n := 0
	for scanner.Scan() {
		token := scanner.Text()
		if token == "" {
			break
		}

		if r.normalize != nil {
			token = r.normalize(token, r.UserContext)
		}

		if r.filter != nil && !r.filter(token, r.UserContext) {
			if r.FailOnInvalid {
				return newErrInvalid(token, n)
			}
			n += len(token)
			continue
		}
		n += len(token)
		s <- token
	}

	if err := scanner.Err(); err != nil && r.FailOnError {
		return newErrRead(err)
	}

	return nil
}

func (r *Reader) createSplitFunc() bufio.SplitFunc {
	if r.delimiter == nil && r.delimiterStr == "" {
		return bufio.ScanLines
	}
	return func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if r.delimiter != nil {
			if loc := r.delimiter.FindIndex(data); loc != nil {
				return loc[1], data[:loc[0]], nil
			}
		} else if r.delimiterStr != "" {
			if prefix, _, found := strings.Cut(string(data), r.delimiterStr); found {
				return len(prefix) + len(r.delimiterStr), []byte(prefix), nil
			}
		} else {
			panic("delimiterStr and delimiter fields are both empty. This should not happen !")
		}

		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}
