package phpgrep

import (
	"strings"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/walker"
)

type compiler struct {
	src []byte
}

func compile(opts *Compiler, pattern []byte, filters []Filter) (*Matcher, error) {
	root, src, err := parsePHP7(pattern)
	if err != nil {
		return nil, err
	}

	if st, ok := root.(*stmt.Expression); ok {
		root = st.Expr
	}

	c := compiler{src: src}
	root.Walk(&c)

	m := &Matcher{m: matcher{root: root}}

	if len(filters) != 0 {
		m.m.filters = map[string][]filterFunc{}
		for _, f := range filters {
			m.m.filters[f.name] = append(m.m.filters[f.name], f.fn)
		}
	}

	return m, nil
}

func (c *compiler) EnterNode(w walker.Walkable) bool {
	if s, ok := w.(*scalar.Encapsed); ok {
		pos := s.GetPosition()
		v := unquoted(string(c.src[pos.StartPos-1 : pos.EndPos]))
		s.Parts = []node.Node{
			&scalar.String{Value: v},
		}
		return true
	}

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
	case "int":
		v.VarName = anyInt{metaNode{name: name}}
	case "float":
		v.VarName = anyFloat{metaNode{name: name}}
	case "str":
		v.VarName = anyStr{metaNode{name: name}}
	case "num":
		v.VarName = anyNum{metaNode{name: name}}
	case "expr":
		v.VarName = anyExpr{metaNode{name: name}}
	}

	return true
}

func (c *compiler) LeaveNode(w walker.Walkable)                  {}
func (c *compiler) EnterChildNode(key string, w walker.Walkable) {}
func (c *compiler) LeaveChildNode(key string, w walker.Walkable) {}
func (c *compiler) EnterChildList(key string, w walker.Walkable) {}
func (c *compiler) LeaveChildList(key string, w walker.Walkable) {}
