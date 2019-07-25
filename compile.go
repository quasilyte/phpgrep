package phpgrep

func compile(opts *Compiler, pattern []byte) (*Matcher, error) {
	root, err := parsePHP7expr(pattern)
	if err != nil {
		return nil, err
	}
	m := &Matcher{m: matcher{root: root}}
	return m, nil
}
