package textio

import "strings"

// s is the string currently being read parameter is set as the [UserContext] attribute
// Used to transform token before passing through the [FilterFunc].
type NormalizeFunc func(s string) string

// Default normalization function. It is a wrapper for the [strings.TrimSpace] function.
func NormalizeTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// This function is a wrapper for the [strings.ToUpper] function.
func NormalizeUpper(s string) string {
	return strings.ToUpper(s)
}

// This function is a wrapper for the [strings.ToLower] function.
func NormalizeLower(s string) string {
	return strings.ToLower(s)
}

// Creates a [NormalizeFunc] function that applies the transformations given by the ns [NormalizeFunc] functions.
// The transformations are applied in the same order as ns.
func ChainNormalizers(ns ...NormalizeFunc) NormalizeFunc {
	return func(s string) string {
		for _, n := range ns {
			s = n(s)
		}
		return s
	}
}
