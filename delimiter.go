package textio

import (
	"bufio"
	"regexp"
	"strings"
)

// By contruction, [regexpr] and [str] cannot be set at the same time.
type Delimiter struct {
	// Delimiter as a regular expression
	regexpr *regexp.Regexp
	// String delimiter
	str string
}

// Default configuration delimiter provider. Default delimiter is "\n" (line-based seperation).
func DefaultDelimiter() *Delimiter {
	return &Delimiter{
		str: "\n",
	}
}

func (d *Delimiter) WithRegexp(regexpr *regexp.Regexp) *Delimiter {
	newD := *d
	newD.SetRegexp(regexpr)
	return &newD
}

func (d *Delimiter) WithStr(s string) *Delimiter {
	newD := *d
	newD.SetStr(s)
	return &newD
}

func (d *Delimiter) WithRegexpFromString(s string) *Delimiter {
	newD := *d
	newD.SetRegexpFromString(s)
	return &newD
}

// Sets the regexpr delimiter.
// This resets the [str] field of `d`.
func (d *Delimiter) SetRegexp(regexpr *regexp.Regexp) {
	if regexpr == nil {
		d.regexpr = regexp.MustCompile("\n")
	} else {

		d.regexpr = regexpr
	}
	d.str = ""
}

// Sets the [str] field of `d` used to seperate input into tokens.
// This resets the [delimiter] field of `d`.
func (d *Delimiter) SetStr(s string) {
	if s == "" {
		s = "\n"
	}
	d.regexpr = nil
	d.str = s
}

// Sets the regexpr delimiter from an expression in string format.
// This resets the [str] field of `d`.
// This function will panic if the expression cannot compile.
func (d *Delimiter) SetRegexpFromString(expr string) {
	regexpr := regexp.MustCompile(expr)
	d.regexpr = regexpr
	d.str = ""
}

func (d *Delimiter) SplitFunc() bufio.SplitFunc {
	return func(data []byte, atEOF bool) (int, []byte, error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if d.regexpr != nil {
			if loc := d.regexpr.FindIndex(data); loc != nil {
				return loc[1], data[:loc[0]], nil
			}
		} else if d.str != "" {
			if prefix, _, found := strings.Cut(string(data), d.str); found {
				return len(prefix) + len(d.str), []byte(prefix), nil
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

func (d *Delimiter) MatchString(s string) bool {
	if d.regexpr != nil {
		return d.regexpr.MatchString(s)
	} else if d.str != "" {
		return strings.Contains(s, d.str)
	} else {
		panic("regexpr and str cannot be both nil !")
	}
}
