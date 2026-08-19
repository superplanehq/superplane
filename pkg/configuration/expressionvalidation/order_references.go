package expressionvalidation

import (
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// ExpressionUsesOrderArtifacts reports whether the expression accesses
// order().artifacts (dot or bracket), including nested uses like
// none(order().artifacts, …). Used to lazy-load artifacts only when needed.
func ExpressionUsesOrderArtifacts(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "artifacts")
}

// ExpressionUsesOrderComments reports whether the expression accesses
// order().comments (dot or bracket), including nested uses like
// len(order().comments). Used to lazy-load comments only when needed.
func ExpressionUsesOrderComments(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "comments")
}

// ExpressionUsesOrderURL reports whether the expression accesses order().url
// (dot or bracket). Used to resolve the work order permalink only when needed,
// since it costs an extra lookup of the factory that owns the order.
func ExpressionUsesOrderURL(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "url")
}

// expressionReferencesOrderProperty reports whether the expression accesses
// order().<property> (dot or bracket), including nested uses such as
// len(order().<property>) or none(order().<property>, …).
func expressionReferencesOrderProperty(expression, property string) (bool, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return false, err
	}

	collector := &orderPropertyCollector{property: property}
	ast.Walk(&tree.Node, collector)
	return collector.found, nil
}

type orderPropertyCollector struct {
	property string
	found    bool
}

func (c *orderPropertyCollector) Visit(node *ast.Node) {
	if c.found {
		return
	}

	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	name, ok := memberPropertyName(member.Property)
	if !ok || name != c.property {
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
