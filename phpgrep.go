package phpgrep

import "regexp"

// Compiler creates matcher objects out of the string patterns.
type Compiler struct {
}

// Compile compiler a given pattern into a matcher.
func (c *Compiler) Compile(pattern []byte, filters ...Filter) (*Matcher, error) {
	return compile(c, pattern, filters)
}

type Filter struct {
	name string
	fn   filterFunc
}

func ValueNotInListFilter(name string, values []string) Filter {
	return Filter{name: name, fn: makeValueNotInListFilter(values)}
}

func ValueInListFilter(name string, values []string) Filter {
	return Filter{name: name, fn: makeValueInListFilter(values)}
}

func RegexpNotFilter(name string, re *regexp.Regexp) Filter {
	return Filter{name: name, fn: makeRegexpNotFilter(re)}
}

func RegexpFilter(name string, re *regexp.Regexp) Filter {
	return Filter{name: name, fn: makeRegexpFilter(re)}
}

// Matcher is a compiled pattern that can be used for PHP code search.
type Matcher struct {
	m matcher
}

type MatchData struct {
	LineFrom int
	LineTo   int

	PosFrom int
	PosTo   int
}

// Match reports whether given PHP code matches the bound pattern.
//
// For malformed inputs (like code with syntax errors), returns false.
func (m *Matcher) Match(code []byte) bool {
	return m.m.match(code)
}

func (m *Matcher) Find(code []byte, callback func(*MatchData) bool) {
	m.m.find(code, callback)
}
