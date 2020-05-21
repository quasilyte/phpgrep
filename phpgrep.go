package phpgrep

import (
	"regexp"

	"github.com/z7zmey/php-parser/node"
)

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

// Clone returns a deep copy of m.
func (m *Matcher) Clone() *Matcher {
	return &Matcher{m: m.m}
}

func (m *Matcher) Find(code []byte, callback func(*MatchData) bool) {
	m.m.find(code, callback)
}

// FindAST is like Find, but accepts parsed node directly.
// Code argument is required to be a source text of the parsed node.
//
// Experimental API!
func (m *Matcher) FindAST(code []byte, root node.Node, callback func(*MatchData) bool) {
	m.m.src = code
	m.m.findAST(root, callback)
}

func (m *Matcher) Match(n node.Node) (MatchData, bool) {
	matched := m.m.match(n)
	return m.m.data, matched
}
