package textio

import (
	"regexp"
	"strings"
)

// s is the string currently being read parameter is set as the [UserContext] attribute.
// Should return true is the token satisfies user defined constraints, false otherwise.
type FilterFunc func(s string) bool

// FilterNonEmpty returns a FilterFunc that rejects empty or whitespace-only strings.
//
// The input string is trimmed using strings.TrimSpace before evaluation.
// If the resulting string is empty, the token is rejected.
func FilterNonEmpty(s string) FilterFunc {
	return func(s string) bool { return strings.TrimSpace(s) != "" }
}

// FilterMinLength returns a FilterFunc that accepts only strings
// whose length is greater than or equal to n.
func FilterMinLength(n int) FilterFunc {
	return func(s string) bool {
		return len(s) >= n
	}
}

// FilterMaxLength returns a FilterFunc that accepts only strings
// whose length is less than or equal to n.
func FilterMaxLength(n int) FilterFunc {
	return func(s string) bool {
		return len(s) <= n
	}
}

// FilterRegexp returns a FilterFunc that accepts strings
// matching the provided regular expression.
//
// The caller is responsible for compiling the regexp.
func FilterRegexp(re *regexp.Regexp) FilterFunc {
	return func(s string) bool {
		return re.MatchString(s)
	}
}

// And combines two FilterFunc using a logical AND.
//
// The resulting filter accepts a string only if both filters
// accept it.
func (f1 FilterFunc) And(f2 FilterFunc) FilterFunc {
	return func(s string) bool {
		return f1(s) && f2(s)
	}
}

// Or combines two FilterFunc using a logical OR.
//
// The resulting filter accepts a string if at least one
// of the filters accepts it.
func (f1 FilterFunc) Or(f2 FilterFunc) FilterFunc {
	return func(s string) bool {
		return f1(s) || f2(s)
	}
}

// Not returns a FilterFunc that negates the result of the given filter.
//
// The resulting filter accepts a string if and only if
// the original filter rejects it.
func Not(f FilterFunc) FilterFunc {
	return func(s string) bool {
		return !f(s)
	}
}
