package phpgrep

// Compiler creates matcher objects out of the string patterns.
type Compiler struct {
}

// Compile compiler a given pattern into a matcher.
func (c *Compiler) Compile(pattern []byte) (*Matcher, error) {
	return compile(c, pattern)
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
