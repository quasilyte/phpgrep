package phpgrep

import (
	"fmt"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/expr/binary"
	"github.com/z7zmey/php-parser/node/expr/cast"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/walker"
)

type matcher struct {
	root node.Node

	src []byte

	handler func(*MatchData) bool
	named   map[string]node.Node
	filters map[string][]filterFunc

	literalMatch bool

	data MatchData
}

func (m *matcher) match(code []byte) bool {
	root, code, err := parsePHP7(code)
	if err != nil {
		return false
	}
	m.src = code
	return m.matchAST(root)
}

func (m *matcher) matchAST(root node.Node) bool {
	matched := false
	m.findAST(root, func(*MatchData) bool {
		matched = true
		return false // Stop at the first match
	})
	return matched
}

func (m *matcher) find(code []byte, callback func(*MatchData) bool) {
	root, code, err := parsePHP7(code)
	if err != nil {
		return
	}
	m.src = code
	m.findAST(root, callback)
}

func (m *matcher) findAST(root node.Node, callback func(*MatchData) bool) {
	m.handler = callback

	root.Walk(m)
}

func (m *matcher) eqNameParts(xs, ys []node.Node) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i, p1 := range xs {
		p1 := p1.(*name.NamePart).Value
		p2 := ys[i].(*name.NamePart).Value
		if p1 != p2 {
			return false
		}
	}
	return true
}

func (m *matcher) eqNodeSliceNoMeta(xs, ys []node.Node) bool {
	if len(xs) != len(ys) {
		return false
	}

	for i, x := range xs {
		if !m.eqNode(x, ys[i]) {
			return false
		}
	}

	return true
}

func (m *matcher) eqNodeSlice(xs, ys []node.Node) bool {
	if len(xs) == 0 && len(ys) != 0 {
		return false
	}

	matchAny := false

	i := 0
	for i < len(xs) {
		x := xs[i]

		if matchMetaVar(x, "*") {
			matchAny = true
		}

		if matchAny {
			switch {
			// "Nothing left to match" stop condition.
			case len(ys) == 0:
				matchAny = false
				i++
			// Lookahead for non-greedy matching.
			case i+1 < len(xs) && m.eqNode(xs[i+1], ys[0]):
				matchAny = false
				i += 2
				ys = ys[1:]
			default:
				ys = ys[1:]
			}
			continue
		}

		if len(ys) == 0 || !m.eqNode(x, ys[0]) {
			return false
		}
		i++
		ys = ys[1:]
	}

	return len(ys) == 0
}

func (m *matcher) eqEncapsedStringPart(x, y node.Node) bool {
	switch x := x.(type) {
	case *scalar.EncapsedStringPart:
		y, ok := y.(*scalar.EncapsedStringPart)
		return ok && x.Value == y.Value
	case *expr.Variable:
		// Match variables literally.
		y, ok := y.(*expr.Variable)
		return ok && m.eqNode(x.VarName, y.VarName)
	default:
		return m.eqNode(x, y)
	}
}

func (m *matcher) eqNode(x, y node.Node) bool {
	if x == y {
		return true
	}

	switch x := x.(type) {
	case nil:
		return y == nil

	case *stmt.Expression:
		// To make it possible to match statements with $-expressions,
		// check whether expression inside x.Expr is a variable.
		if x, ok := x.Expr.(*expr.Variable); ok {
			return m.eqVariable(x, y)
		}
		y, ok := y.(*stmt.Expression)
		return ok && m.eqNode(x.Expr, y.Expr)

	case *stmt.StmtList:
		y, ok := y.(*stmt.StmtList)
		return ok && m.eqNodeSlice(x.Stmts, y.Stmts)

	case *stmt.Nop:
		_, ok := y.(*stmt.Nop)
		return ok
	case *stmt.While:
		y, ok := y.(*stmt.While)
		return ok && m.eqNode(x.Cond, y.Cond) && m.eqNode(x.Stmt, y.Stmt)

	case *stmt.Else:
		y, ok := y.(*stmt.Else)
		return ok && m.eqNode(x.Stmt, y.Stmt)
	case *stmt.ElseIf:
		y, ok := y.(*stmt.ElseIf)
		return ok && m.eqNode(x.Cond, y.Cond) && m.eqNode(x.Stmt, y.Stmt)
	case *stmt.If:
		y, ok := y.(*stmt.If)
		return ok && m.eqNodeSliceNoMeta(x.ElseIf, y.ElseIf) &&
			m.eqNode(x.Cond, y.Cond) &&
			m.eqNode(x.Stmt, y.Stmt) &&
			m.eqNode(x.Else, y.Else)

	case *expr.List:
		y, ok := y.(*expr.List)
		return ok && m.eqNodeSlice(x.Items, y.Items)

	case *expr.New:
		y, ok := y.(*expr.New)
		if !ok || !m.eqNode(x.Class, y.Class) {
			return false
		}
		if x.ArgumentList == nil || y.ArgumentList == nil {
			return x.ArgumentList == y.ArgumentList
		}
		return m.eqNodeSlice(x.ArgumentList.Arguments, y.ArgumentList.Arguments)

	case *stmt.Case:
		y, ok := y.(*stmt.Case)
		return ok && m.eqNode(x.Cond, y.Cond) && m.eqNodeSlice(x.Stmts, y.Stmts)
	case *stmt.Default:
		y, ok := y.(*stmt.Default)
		return ok && m.eqNodeSlice(x.Stmts, y.Stmts)
	case *stmt.Switch:
		y, ok := y.(*stmt.Switch)
		return ok && m.eqNode(x.Cond, y.Cond) &&
			m.eqNodeSlice(x.CaseList.Cases, y.CaseList.Cases)

	case *stmt.Return:
		y, ok := y.(*stmt.Return)
		return ok && m.eqNode(x.Expr, y.Expr)

	case *assign.Assign:
		y, ok := y.(*assign.Assign)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Plus:
		y, ok := y.(*assign.Plus)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Reference:
		y, ok := y.(*assign.Reference)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.BitwiseAnd:
		y, ok := y.(*assign.BitwiseAnd)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.BitwiseOr:
		y, ok := y.(*assign.BitwiseOr)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.BitwiseXor:
		y, ok := y.(*assign.BitwiseXor)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Concat:
		y, ok := y.(*assign.Concat)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Div:
		y, ok := y.(*assign.Div)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Minus:
		y, ok := y.(*assign.Minus)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Mod:
		y, ok := y.(*assign.Mod)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Mul:
		y, ok := y.(*assign.Mul)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.Pow:
		y, ok := y.(*assign.Pow)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.ShiftLeft:
		y, ok := y.(*assign.ShiftLeft)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)
	case *assign.ShiftRight:
		y, ok := y.(*assign.ShiftRight)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Expression, y.Expression)

	case *expr.ArrayDimFetch:
		y, ok := y.(*expr.ArrayDimFetch)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Dim, y.Dim)
	case *expr.ArrayItem:
		y, ok := y.(*expr.ArrayItem)
		if !ok {
			return false
		}
		if x.Key == nil || y.Key == nil {
			return x.Key == y.Key && m.eqNode(x.Val, y.Val)
		}
		return m.eqNode(x.Key, y.Key) && m.eqNode(x.Val, y.Val)
	case *expr.ShortArray:
		y, ok := y.(*expr.ShortArray)
		return ok && m.eqNodeSlice(x.Items, y.Items)
	case *expr.Array:
		y, ok := y.(*expr.Array)
		return ok && m.eqNodeSlice(x.Items, y.Items)

	case *node.Argument:
		y, ok := y.(*node.Argument)
		return ok && x.IsReference == y.IsReference &&
			x.Variadic == y.Variadic &&
			m.eqNode(x.Expr, y.Expr)
	case *expr.FunctionCall:
		y, ok := y.(*expr.FunctionCall)
		if !ok || !m.eqNode(x.Function, y.Function) {
			return false
		}
		return m.eqNodeSlice(x.ArgumentList.Arguments, y.ArgumentList.Arguments)

	case *expr.PostInc:
		y, ok := y.(*expr.PostInc)
		return ok && m.eqNode(x.Variable, y.Variable)
	case *expr.PostDec:
		y, ok := y.(*expr.PostDec)
		return ok && m.eqNode(x.Variable, y.Variable)
	case *expr.PreInc:
		y, ok := y.(*expr.PreInc)
		return ok && m.eqNode(x.Variable, y.Variable)
	case *expr.PreDec:
		y, ok := y.(*expr.PreDec)
		return ok && m.eqNode(x.Variable, y.Variable)

	case *expr.Exit:
		y, ok := y.(*expr.Exit)
		return ok && x.Die == y.Die && m.eqNode(x.Expr, y.Expr)

	case *expr.Include:
		y, ok := y.(*expr.Include)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.IncludeOnce:
		y, ok := y.(*expr.IncludeOnce)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.Require:
		y, ok := y.(*expr.Require)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.RequireOnce:
		y, ok := y.(*expr.RequireOnce)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.Empty:
		y, ok := y.(*expr.Empty)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.Eval:
		y, ok := y.(*expr.Eval)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.ErrorSuppress:
		y, ok := y.(*expr.ErrorSuppress)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.Clone:
		y, ok := y.(*expr.Clone)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.BitwiseNot:
		y, ok := y.(*expr.BitwiseNot)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.BooleanNot:
		y, ok := y.(*expr.BooleanNot)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.UnaryMinus:
		y, ok := y.(*expr.UnaryMinus)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *expr.UnaryPlus:
		y, ok := y.(*expr.UnaryPlus)
		return ok && m.eqNode(x.Expr, y.Expr)

	case *expr.StaticPropertyFetch:
		switch y := y.(type) {
		case *expr.StaticPropertyFetch:
			return m.eqNode(x.Class, y.Class) &&
				m.eqNode(x.Property, y.Property)
		case *expr.ClassConstFetch:
			return nodeIsVar(x.Property) &&
				m.eqNode(x.Class, y.Class) &&
				m.eqNode(x.Property, y.ConstantName)
		default:
			return false
		}

	case *expr.ClassConstFetch:
		y, ok := y.(*expr.ClassConstFetch)
		return ok && m.eqNode(x.Class, y.Class) && m.eqNode(x.ConstantName, y.ConstantName)
	case *expr.StaticCall:
		y, ok := y.(*expr.StaticCall)
		return ok &&
			m.eqNode(x.Class, y.Class) &&
			m.eqNode(x.Call, y.Call) &&
			m.eqNodeSlice(x.ArgumentList.Arguments, y.ArgumentList.Arguments)

	case *scalar.Encapsed:
		y, ok := y.(*scalar.Encapsed)
		if !ok {
			return false
		}
		if len(x.Parts) != len(y.Parts) {
			return false
		}
		for i, px := range x.Parts {
			if !m.eqEncapsedStringPart(px, y.Parts[i]) {
				return false
			}
		}
		return true

	case *scalar.MagicConstant:
		y, ok := y.(*scalar.MagicConstant)
		return ok && y.Value == x.Value
	case *scalar.Lnumber:
		y, ok := y.(*scalar.Lnumber)
		return ok && y.Value == x.Value
	case *scalar.Dnumber:
		y, ok := y.(*scalar.Dnumber)
		return ok && y.Value == x.Value
	case *scalar.String:
		y, ok := y.(*scalar.String)
		return ok && y.Value == x.Value

	case *binary.Coalesce:
		y, ok := y.(*binary.Coalesce)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Concat:
		y, ok := y.(*binary.Concat)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Div:
		y, ok := y.(*binary.Div)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Mod:
		y, ok := y.(*binary.Mod)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Mul:
		y, ok := y.(*binary.Mul)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Pow:
		y, ok := y.(*binary.Pow)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.BitwiseAnd:
		y, ok := y.(*binary.BitwiseAnd)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.BitwiseOr:
		y, ok := y.(*binary.BitwiseOr)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.BitwiseXor:
		y, ok := y.(*binary.BitwiseXor)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.ShiftLeft:
		y, ok := y.(*binary.ShiftLeft)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.ShiftRight:
		y, ok := y.(*binary.ShiftRight)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.LogicalAnd:
		y, ok := y.(*binary.LogicalAnd)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.LogicalOr:
		y, ok := y.(*binary.LogicalOr)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.LogicalXor:
		y, ok := y.(*binary.LogicalXor)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.BooleanAnd:
		y, ok := y.(*binary.BooleanAnd)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.BooleanOr:
		y, ok := y.(*binary.BooleanOr)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.NotEqual:
		y, ok := y.(*binary.NotEqual)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.NotIdentical:
		y, ok := y.(*binary.NotIdentical)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Equal:
		y, ok := y.(*binary.Equal)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Identical:
		y, ok := y.(*binary.Identical)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Greater:
		y, ok := y.(*binary.Greater)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.GreaterOrEqual:
		y, ok := y.(*binary.GreaterOrEqual)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Smaller:
		y, ok := y.(*binary.Smaller)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.SmallerOrEqual:
		y, ok := y.(*binary.SmallerOrEqual)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Spaceship:
		y, ok := y.(*binary.Spaceship)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Plus:
		y, ok := y.(*binary.Plus)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)
	case *binary.Minus:
		y, ok := y.(*binary.Minus)
		return ok && m.eqNode(x.Left, y.Left) && m.eqNode(x.Right, y.Right)

	case *expr.ConstFetch:
		y, ok := y.(*expr.ConstFetch)
		return ok && m.eqNode(x.Constant, y.Constant)
	case *name.Name:
		y, ok := y.(*name.Name)
		return ok && m.eqNameParts(x.Parts, y.Parts)
	case *name.FullyQualified:
		y, ok := y.(*name.FullyQualified)
		return ok && m.eqNameParts(x.Parts, y.Parts)
	case *node.Identifier:
		y, ok := y.(*node.Identifier)
		return ok && x.Value == y.Value
	case *expr.Variable:
		return m.eqVariable(x, y)

	case *expr.Ternary:
		y, ok := y.(*expr.Ternary)
		return ok && m.eqNode(x.Condition, y.Condition) &&
			m.eqNode(x.IfTrue, y.IfTrue) &&
			m.eqNode(x.IfFalse, y.IfFalse)

	case *expr.Isset:
		y, ok := y.(*expr.Isset)
		return ok && m.eqNodeSlice(x.Variables, y.Variables)

	case *expr.PropertyFetch:
		y, ok := y.(*expr.PropertyFetch)
		return ok && m.eqNode(x.Variable, y.Variable) && m.eqNode(x.Property, y.Property)
	case *expr.MethodCall:
		y, ok := y.(*expr.MethodCall)
		return ok && m.eqNode(x.Variable, y.Variable) &&
			m.eqNode(x.Method, y.Method) &&
			m.eqNodeSlice(x.ArgumentList.Arguments, y.ArgumentList.Arguments)

	case *cast.Double:
		y, ok := y.(*cast.Double)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *cast.Array:
		y, ok := y.(*cast.Array)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *cast.Bool:
		y, ok := y.(*cast.Bool)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *cast.Int:
		y, ok := y.(*cast.Int)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *cast.Object:
		y, ok := y.(*cast.Object)
		return ok && m.eqNode(x.Expr, y.Expr)
	case *cast.String:
		y, ok := y.(*cast.String)
		return ok && m.eqNode(x.Expr, y.Expr)

	default:
		panic(fmt.Sprintf("(??) %T %T\n", x, y))
	}
}

func (m *matcher) matchNamed(name string, y node.Node) bool {
	// Special case.
	// "_" name matches anything, always.
	// Anonymous names replaced with "_" during the compilation.
	if name == "_" {
		return true
	}

	// FIXME: a hack to avoid matching a root against $x.
	if _, ok := y.(*node.Root); ok {
		return false
	}

	z, ok := m.named[name]
	if !ok {
		filters := m.filters[name]
		if len(filters) == 0 {
			m.named[name] = y
			return true
		}
		pos := y.GetPosition()
		buf := m.src[pos.StartPos-1 : pos.EndPos]
		for _, filter := range filters {
			if !filter(buf) {
				return false
			}
		}
		m.named[name] = y
		return true
	}
	if z == nil {
		return y == nil
	}

	m.literalMatch = true
	result := m.eqNode(z, y)
	m.literalMatch = false
	return result
}

func (m *matcher) eqVariable(x *expr.Variable, y node.Node) bool {
	if m.literalMatch {
		y, ok := y.(*expr.Variable)
		return ok && m.eqNode(x.VarName, y.VarName)
	}

	switch vn := x.VarName.(type) {
	case *node.Identifier:
		return m.matchNamed(vn.Value, y)
	case anyVar:
		_, ok := y.(*expr.Variable)
		return ok && m.matchNamed(vn.name, y)
	case anyInt:
		_, ok := y.(*scalar.Lnumber)
		return ok && m.matchNamed(vn.name, y)
	case anyFloat:
		_, ok := y.(*scalar.Dnumber)
		return ok && m.matchNamed(vn.name, y)
	case anyStr:
		_, ok := y.(*scalar.String)
		return ok && m.matchNamed(vn.name, y)
	case anyNum:
		switch y.(type) {
		case *scalar.Lnumber, *scalar.Dnumber:
			return m.matchNamed(vn.name, y)
		default:
			return false
		}
	case anyExpr:
		return nodeIsExpr(y) && m.matchNamed(vn.name, y)
	}

	if y, ok := y.(*expr.Variable); ok {
		return m.eqNode(x.VarName, y.VarName)
	}
	return false
}

func (m *matcher) EnterNode(w walker.Walkable) bool {
	n, ok := w.(node.Node)
	if !ok {
		return true
	}

	m.named = map[string]node.Node{}

	if ok && m.eqNode(m.root, n) {
		pos := n.GetPosition()
		if pos == nil {
			// FIXME: is there a more elegant way to endure things like [,,]
			// without pos?
			if n, ok := n.(*expr.ArrayItem); ok && n.Key == nil && n.Val == nil {
				return true
			}
			panic(fmt.Sprintf("nil pos for %q node (%T)", nodeString(n), n))
		}
		m.data.LineFrom = pos.StartLine
		m.data.LineTo = pos.EndLine
		m.data.PosFrom = pos.StartPos - 1
		m.data.PosTo = pos.EndPos

		return m.handler(&m.data)
	}

	return true
}

func (m *matcher) LeaveNode(w walker.Walkable)                  {}
func (m *matcher) EnterChildNode(key string, w walker.Walkable) {}
func (m *matcher) LeaveChildNode(key string, w walker.Walkable) {}
func (m *matcher) EnterChildList(key string, w walker.Walkable) {}
func (m *matcher) LeaveChildList(key string, w walker.Walkable) {}
