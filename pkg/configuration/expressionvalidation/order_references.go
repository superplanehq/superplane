package expressionvalidation

import (
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// ExpressionUsesOrderArtifacts reports whether the expression accesses
// order().artifacts (dot or bracket), including nested uses like
// none(order().artifacts, …). Used to lazy-load artifacts only when needed.
func ExpressionUsesOrderArtifacts(expression string) (bool, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return false, err
	}

	collector := &orderArtifactsCollector{}
	ast.Walk(&tree.Node, collector)
	return collector.found, nil
}

type orderArtifactsCollector struct {
	found bool
}

func (c *orderArtifactsCollector) Visit(node *ast.Node) {
	if c.found {
		return
	}

	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	name, ok := memberPropertyName(member.Property)
	if !ok || name != "artifacts" {
		return
	}

	if isOrderCall(member.Node) {
		c.found = true
	}
}

func memberPropertyName(property ast.Node) (string, bool) {
	switch prop := property.(type) {
	case *ast.StringNode:
		return prop.Value, true
	case *ast.IdentifierNode:
		return prop.Value, true
	default:
		return "", false
	}
}

func isOrderCall(node ast.Node) bool {
	call, ok := node.(*ast.CallNode)
	if !ok {
		return false
	}

	ident, ok := call.Callee.(*ast.IdentifierNode)
	return ok && ident.Value == "order"
}
