package phpgrep

import (
	"bytes"
	"errors"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/printer"
)

func unquoted(s string) string {
	return s[1 : len(s)-1]
}

func matchMetaVar(n node.Node, s string) bool {
	switch n := n.(type) {
	case *expr.ArrayItem:
		return n.Key == nil && matchMetaVar(n.Val, s)
	case *stmt.Expression:
		return matchMetaVar(n.Expr, s)
	case *node.Argument:
		return matchMetaVar(n.Expr, s)

	case *expr.Variable:
		nm, ok := n.VarName.(*scalar.String)
		return ok && unquoted(nm.Value) == s

	default:
		return false
	}
}

func nodeString(n node.Node) string {
	var b bytes.Buffer
	printer.NewPrettyPrinter(&b, " ").Print(n)
	return b.String()
}

func parsePHP7(code []byte) (node.Node, error) {
	if bytes.HasPrefix(code, []byte("<?")) || bytes.HasPrefix(code, []byte("<?php")) {
		return parsePHP7root(code)
	}
	return parsePHP7expr(code)
}

func parsePHP7expr(code []byte) (node.Node, error) {
	code = append([]byte("<?php "), code...)
	code = append(code, ';')
	root, err := parsePHP7root(code)
	if err != nil {
		return nil, err
	}
	return root.(*node.Root).Stmts[0], nil
}

func parsePHP7root(code []byte) (node.Node, error) {
	p := php7.NewParser(bytes.NewReader(code), "string-input.php")
	p.Parse()
	if len(p.GetErrors()) != 0 {
		return nil, errors.New(p.GetErrors()[0].String())
	}
	return p.GetRootNode(), nil
}
