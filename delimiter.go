package textio

import (
	"bufio"
	"bytes"
	"regexp"
)

type Delimiter struct {
	token pattern
	stop  pattern
}

// By contruction, [regexpr] and [str] cannot be set at the same time.
type pattern struct {
	// Delimiter as a regular expression
	re *regexp.Regexp
	// String delimiter
	str string
}

// Default configuration delimiter provider. Default delimiter is "\n" (line-based seperation).
func NewDelimiter() *Delimiter {
	return DefaultDelimiter()
}

func DefaultDelimiter() *Delimiter {
	return &Delimiter{
		token: pattern{str: "\n"},
		stop:  pattern{str: "\n\n"},
	}
}

// Sets the regexpr delimiter.
// This resets the [str] field of `d`.
func (d *Delimiter) SetTokenRegexp(regexpr *regexp.Regexp) {
	d.token.re = regexpr
	d.token.str = ""
}

// Sets the [str] field of `d` used to seperate input into tokens.
// This resets the [delimiter] field of `d`.
func (d *Delimiter) SetTokenStr(s string) {
	d.token.re = nil
	d.token.str = s
}

// Sets the regexpr delimiter from an expression in string format.
// This resets the [str] field of `d`.
// This function will panic if the expression cannot compile.
func (d *Delimiter) SetTokenRegexpFromString(expr string) {
	if expr == "" {
		panic("empty regexp is not allowed")
	}
	regexpr := regexp.MustCompile(expr)
	d.token.re = regexpr
	d.token.str = ""
}

// Sets the regexpr delimiter.
// This resets the [str] field of `d`.
func (d *Delimiter) SetStopRegexp(regexpr *regexp.Regexp) {
	d.stop.re = regexpr
	d.stop.str = ""
}

// Sets the [str] field of `d` used to seperate input into tokens.
// This resets the [delimiter] field of `d`.
func (d *Delimiter) SetStopStr(s string) {
	d.stop.re = nil
	d.stop.str = s
}

// Sets the regexpr delimiter from an expression in string format.
// This resets the [str] field of `d`.
// This function will panic if the expression cannot compile.
func (d *Delimiter) SetStopRegexpFromString(expr string) {
	if expr == "" {
		panic("empty regexp is not allowed")
	}
	regexpr := regexp.MustCompile(expr)
	d.stop.re = regexpr
	d.stop.str = ""
}

func (d Delimiter) WithTokenRegexp(regexpr *regexp.Regexp) *Delimiter {
	d.token = pattern{re: regexpr}
	return &d
}

func (d Delimiter) WithTokenStr(s string) *Delimiter {
	d.token = pattern{str: s}
	return &d
}

func (d Delimiter) WithTokenRegexpFromString(s string) *Delimiter {
	if s == "" {
		panic("empty regexp is not allowed")
	}
	d.token = pattern{re: regexp.MustCompile(s)}
	return &d
}

func (d Delimiter) WithStopRegexp(regexpr *regexp.Regexp) *Delimiter {
	d.stop = pattern{re: regexpr}
	return &d
}

func (d Delimiter) WithStopStr(s string) *Delimiter {
	d.stop = pattern{str: s}
	return &d
}

func (d Delimiter) WithStopRegexpFromString(s string) *Delimiter {
	if s == "" {
		panic("empty regexp is not allowed")
	}

	d.stop = pattern{re: regexp.MustCompile(s)}
	return &d
}

func (d *Delimiter) SplitFunc() bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {

		// Nothing left
		if atEOF && len(data) == 0 {
			return 0, nil, bufio.ErrFinalToken
		}

		// Locate delimiters
		tokenIdx, tokenW := d.token.find(data)

		stopIdx, stopW := -1, 0
		if d.stop.enabled() {
			stopIdx, stopW = d.stop.find(data)
		}

		if stopIdx >= 0 && (tokenIdx < 0 || stopIdx < tokenIdx) {
			// Return data before stop as final token
			if stopIdx > 0 {
				return stopIdx, data[:stopIdx], nil
			}

			// Stop delimiter at beginning: consume and stop
			return stopW, nil, bufio.ErrFinalToken
		}

		if tokenIdx >= 0 {
			return tokenIdx + tokenW, data[:tokenIdx], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		// Need more data
		return 0, nil, nil
	}
}

func (p pattern) enabled() bool {
	return p.re != nil || p.str != ""
}

func (p *pattern) find(data []byte) (idx int, width int) {
	if p == nil {
		return -1, 0
	}

	if p.re != nil {
		loc := p.re.FindIndex(data)
		if loc == nil {
			return -1, 0
		}
		return loc[0], loc[1] - loc[0]
	}

	if p.str != "" {
		idx := bytes.Index(data, []byte(p.str))
		if idx < 0 {
			return -1, 0
		}
		return idx, len(p.str)
	}

	return -1, 0
}
