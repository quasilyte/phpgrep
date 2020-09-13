package phpgrep

import (
	"text/template/parse"
)

type formatDeps struct {
	capture bool
}

func inspectFormatDeps(format string) formatDeps {
	var deps formatDeps

	treeMap, err := parse.Parse("output-format", format, "", "", nil)
	if err != nil {
		return deps
	}
	tree, ok := treeMap["output-format"]
	if !ok {
		return deps
	}

	walkTemplate(tree.Root, func(n parse.Node) bool {
		switch n := n.(type) {
		case *parse.FieldNode:
			if len(n.Ident) != 1 {
				break
			}
			switch n.Ident[0] {
			case "Filename", "Line", "Match":
				// No need to track these.
			default:
				deps.capture = true
			}
		}
		return true
	})

	return deps
}

func walkTemplate(n parse.Node, visit func(parse.Node) bool) {
	if n == nil {
		return
	}
	if !visit(n) {
		return
	}

	switch n := n.(type) {
	case *parse.ListNode:
		walkTemplateSlice(n.Nodes, visit)

	case *parse.PipeNode:
		for i := range n.Decl {
			walkTemplate(n.Decl[i], visit)
		}
		for i := range n.Cmds {
			walkTemplate(n.Cmds[i], visit)
		}

	case *parse.ActionNode:
		walkTemplate(n.Pipe, visit)

	case *parse.CommandNode:
		walkTemplateSlice(n.Args, visit)

	case *parse.ChainNode:
		walkTemplate(n.Node, visit)

	case *parse.IfNode:
		walkTemplateBranch(n.BranchNode, visit)

	case *parse.RangeNode:
		walkTemplateBranch(n.BranchNode, visit)

	case *parse.TemplateNode:
		walkTemplate(n.Pipe, visit)
	}
}

func walkTemplateBranch(n parse.BranchNode, visit func(parse.Node) bool) {
	walkTemplate(n.Pipe, visit)
	walkTemplate(n.List, visit)
	if n.ElseList != nil {
		walkTemplate(n.ElseList, visit)
	}
}

func walkTemplateSlice(nodes []parse.Node, visit func(parse.Node) bool) {
	for _, n := range nodes {
		walkTemplate(n, visit)
	}
}
