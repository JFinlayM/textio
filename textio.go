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
//
// Error handling is configurable: the caller may choose to fail immediately
// on errors or invalid input, or to continue processing depending on the
// Reader configuration.
//
// This package is intentionally low-level and does not impose any specific
// text model or output structure, leaving higher-level semantics to the user.
package textio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// s is the string currently being read, ctx parameter is set as the [UserContext] attribute
// Used to transform token before passing through the [FilterFunc].
type NormalizeFunc func(s string, ctx any) string

// s is the string currently being read, ctx parameter is set as the [UserContext] attribute.
// Should return true is the token satisfies user defined constraints, false otherwise.
type FilterFunc func(s string, ctx any) bool

// ErrorFormatter can be implemented to customize how errors are created
// and returned by the textio package.
//
// When provided to a Reader, ErrorFormatter is used instead of the standard
// error constructors to build errors originating from
// scanning, normalization, or filtering failures.
//
// This allows users to:
//   - wrap errors with additional context
//   - attach custom error types
//   - integrate with application-specific error handling logic
//
// If no ErrorFormatter is set, [textio] falls back to returning the original
// error or the standard formatted error [fmt.Errorf].
type ErrorFormatter interface {
	// Errorf formats an error according to a format specifier.
	Errorf(format string, args ...any) error

	// Error transforms or wraps an existing error.
	Error(err error) error
}

// [Reader] reads tokens from an io.Reader and optionally applies
// normalization and filtering before returning them.
//
// [Reader] supports both batch and streaming consumption patterns
// and can be configured to control error handling behavior.
type Reader struct {
	reader         io.Reader
	delimiter      string
	normalize      NormalizeFunc
	filter         FilterFunc
	UserContext    any
	FailOnError    bool
	FailOnInvalid  bool
	errorFormatter ErrorFormatter
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
		reader:      os.Stdin,
		delimiter:   "\n",
		normalize:   DefaultNormalizer,
		FailOnError: true,
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

// Sets the delimiter used to seperate input into tokens.
func (r *Reader) SetDelimiter(s string) {
	if s == "" {
		s = "\n"
	}
	r.delimiter = s
}

// Sets the function to be called to normalize current read token before passing through filter function. There is none by default.
func (r *Reader) SetNormalizer(normalizeFunc NormalizeFunc) {
	r.normalize = normalizeFunc
}

// Sets the function to be called to filter current read token. Should return true is the token satisfies user defined constraints, false otherwise.
func (r *Reader) SetFilter(filterFunc FilterFunc) {
	r.filter = filterFunc
}

// Sets the error formatter in the r [Reader] structure.
func (r *Reader) SetErrorFormatter(errorFormatter ErrorFormatter) {
	r.errorFormatter = errorFormatter
}

// Read processes input from the provided [io.Reader](s).
// It reads strings, applies normalization and filtering if specified,
// and returns the resulting strings or an error if any issues occur.
//
// Returns:
//   - A slice of strings containing the processed input.
//   - An error if any issues occur during reading or processing, depending on the configuration.
//
// Behavior:
//   - If a delimiter is specified in the [Reader], it uses a custom split function
//     to tokenize the input; otherwise, it defaults to line-based scanning.
//   - If a normalization function is provided, it applies the function to each string read.
//   - If a filtering function is provided, it validates each string against the filter.
//     If a string fails the filter and FailOnInvalid is true, the function returns an error. Otherwise, it skips the invalid string.
//   - If an error occurs during scanning and FailOnError is true, the function returns the error.
func (r *Reader) ReadAll() ([]string, error) {
	var parts []string

	scanner := bufio.NewScanner(r.reader)

	if r.delimiter != "" {
		scanner.Split(r.createSplitFunc())
	}

	for scanner.Scan() {
		part := scanner.Text()
		if part == "" {
			break
		}
		if r.normalize != nil {
			part = r.normalize(part, r.UserContext)
		}

		if r.filter != nil && !r.filter(part, r.UserContext) {
			if r.FailOnInvalid {
				return parts, r.errorf("String '%s' didn't respect constraints\n", part)
			}
			continue
		}

		parts = append(parts, part)
	}

	if err := scanner.Err(); err != nil && r.FailOnError {
		return parts, r.error(err)
	}

	return parts, nil
}

// Read processes input from the provided [io.Reader](s).
// It populates 0 <= n <= len(p) bytes from the files in p,
// and returns an error if any issues occur.
//
// Returns:
//   - n: number of bytes read
//   - err: an error if any issues occur during reading
func (r *Reader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Read processes input from the provided [io.Reader](s).
// It reads strings, applies normalization and filtering if specified.
// The resulting strings are passed through the `s` channel or an error is returned if any issues occur.
//
// Parameters:
//   - s: the string channel
//
// Returns:
//   - An error if any issues occur during reading or processing, depending on the configuration.
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

	if r.delimiter != "" {
		scanner.Split(r.createSplitFunc())
	}

	for scanner.Scan() {
		part := scanner.Text()
		if part == "" {
			break
		}

		if r.normalize != nil {
			part = r.normalize(part, r.UserContext)
		}

		if r.filter != nil && !r.filter(part, r.UserContext) {
			if r.FailOnInvalid {
				return r.errorf("String '%s' didn't respect constraints\n", part)
			}
			continue
		}

		s <- part
	}

	if err := scanner.Err(); err != nil && r.FailOnError {
		return r.error(err)
	}

	return nil
}

func (r *Reader) createSplitFunc() bufio.SplitFunc {
	return func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if prefix, _, found := strings.Cut(string(data), r.delimiter); found {
			return len(prefix) + len(r.delimiter), []byte(prefix), nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}

func (r *Reader) errorf(format string, a ...any) error {
	if r.errorFormatter != nil {
		return r.errorFormatter.Errorf(format, a...)
	} else {
		return fmt.Errorf(format, a...)
	}
}

func (r *Reader) error(err error) error {
	if r.errorFormatter != nil {
		return r.errorFormatter.Error(err)
	} else {
		return err
	}
}
