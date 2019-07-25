package phpgrep

import (
	"strings"

	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/walker"
)

type compiler struct{}

func compile(opts *Compiler, pattern []byte) (*Matcher, error) {
	root, err := parsePHP7expr(pattern)
	if err != nil {
		return nil, err
	}

	var c compiler
	root.Walk(&c)

	m := &Matcher{m: matcher{root: root}}
	return m, nil
}

func (c *compiler) EnterNode(w walker.Walkable) bool {
	v, ok := w.(*expr.Variable)
	if !ok {
		return true
	}
	s, ok := v.VarName.(*scalar.String)
	if !ok {
		return true
	}
	value := unquoted(s.Value)

	var name string
	var class string

	colon := strings.Index(value, ":")
	if colon == -1 {
		// Anonymous matcher.
		name = "_"
		class = value
	} else {
		// Named matcher.
		name = value[:colon]
		class = value[colon+len(":"):]
	}

	switch class {
	case "var":
		v.VarName = anyVar{metaNode{name: name}}
	}

	return true
}

func (c *compiler) LeaveNode(w walker.Walkable)                  {}
func (c *compiler) EnterChildNode(key string, w walker.Walkable) {}
func (c *compiler) LeaveChildNode(key string, w walker.Walkable) {}
func (c *compiler) EnterChildList(key string, w walker.Walkable) {}
func (c *compiler) LeaveChildList(key string, w walker.Walkable) {}
