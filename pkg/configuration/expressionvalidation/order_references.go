package expressionvalidation

import (
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// ExpressionUsesOrderArtifacts reports whether the expression accesses
// order().artifacts or task().artifacts (dot or bracket), including nested
// uses like none(order().artifacts, …). Used to lazy-load artifacts only when needed.
func ExpressionUsesOrderArtifacts(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "artifacts")
}

// ExpressionUsesOrderComments reports whether the expression accesses
// order().comments or task().comments (dot or bracket), including nested uses
// like len(order().comments). Used to lazy-load comments only when needed.
func ExpressionUsesOrderComments(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "comments")
}

func ExpressionUsesOrderPullRequests(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "pullRequests")
}

func ExpressionUsesOrderAssignees(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "assignees")
}

// ExpressionUsesOrderURL reports whether the expression accesses order().url
// or task().url (dot or bracket). Used to resolve the work order permalink
// only when needed, since it costs an extra lookup of the factory that owns
// the order.
func ExpressionUsesOrderURL(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "url")
}

// ExpressionUsesOrderKey reports whether the expression accesses order().key
// or task().key (dot or bracket). Used to resolve the work order display
// key (e.g. "SUPER-70") only when needed, since it costs an extra lookup of
// the factory that owns the order.
func ExpressionUsesOrderKey(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "key")
}

// ExpressionUsesOrderSpec reports whether the expression accesses order().spec
// or task().spec (dot or bracket). Used to load the refinement spec only when
// the implement prompt asks for it.
func ExpressionUsesOrderSpec(expression string) (bool, error) {
	return expressionReferencesOrderProperty(expression, "spec")
}

// expressionReferencesOrderProperty reports whether the expression accesses
// order().<property> or task().<property> (dot or bracket), including nested
// uses such as len(order().<property>) or none(task().<property>, …).
func expressionReferencesOrderProperty(expression, property string) (bool, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return false, err
	}

	aliases := collectOrderAliases(tree.Node)
	collector := &orderPropertyCollector{property: property, aliases: aliases}
	ast.Walk(&tree.Node, collector)
	return collector.found, nil
}

func collectOrderAliases(node ast.Node) map[string]struct{} {
	collector := &orderAliasCollector{aliases: map[string]struct{}{}}
	ast.Walk(&node, collector)
	return collector.aliases
}

type orderAliasCollector struct {
	aliases map[string]struct{}
}

func (c *orderAliasCollector) Visit(node *ast.Node) {
	decl, ok := (*node).(*ast.VariableDeclaratorNode)
	if !ok {
		return
	}
	if isOrderCall(decl.Value) {
		c.aliases[decl.Name] = struct{}{}
	}
}

type orderPropertyCollector struct {
	property string
	aliases  map[string]struct{}
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

	if isOrderSource(member.Node, c.aliases) {
		c.found = true
	}
}

func isOrderSource(node ast.Node, aliases map[string]struct{}) bool {
	if isOrderCall(node) {
		return true
	}
	ident, ok := node.(*ast.IdentifierNode)
	if !ok {
		return false
	}
	_, found := aliases[ident.Value]
	return found
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
	return ok && (ident.Value == "order" || ident.Value == "task")
}
