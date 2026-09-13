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

	return referencesOrderProperty(tree.Node, property, nil), nil
}

func referencesOrderProperty(node ast.Node, property string, aliases map[string]struct{}) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case *ast.VariableDeclaratorNode:
		inner := inheritOrderAliases(aliases)
		if isOrderCall(n.Value) {
			inner[n.Name] = struct{}{}
		} else {
			delete(inner, n.Name)
		}
		return referencesOrderProperty(n.Value, property, aliases) ||
			referencesOrderProperty(n.Expr, property, inner)
	case *ast.MemberNode:
		if name, ok := memberPropertyName(n.Property); ok && name == property && isOrderSource(n.Node, aliases) {
			return true
		}
		return referencesOrderProperty(n.Node, property, aliases) ||
			referencesOrderProperty(n.Property, property, aliases)
	case *ast.UnaryNode:
		return referencesOrderProperty(n.Node, property, aliases)
	case *ast.BinaryNode:
		return referencesOrderProperty(n.Left, property, aliases) ||
			referencesOrderProperty(n.Right, property, aliases)
	case *ast.ChainNode:
		return referencesOrderProperty(n.Node, property, aliases)
	case *ast.SliceNode:
		return referencesOrderProperty(n.Node, property, aliases) ||
			referencesOrderProperty(n.From, property, aliases) ||
			referencesOrderProperty(n.To, property, aliases)
	case *ast.CallNode:
		if referencesOrderProperty(n.Callee, property, aliases) {
			return true
		}
		return referencesOrderPropertyList(n.Arguments, property, aliases)
	case *ast.BuiltinNode:
		return referencesOrderPropertyList(n.Arguments, property, aliases)
	case *ast.PredicateNode:
		return referencesOrderProperty(n.Node, property, aliases)
	case *ast.SequenceNode:
		return referencesOrderPropertyList(n.Nodes, property, aliases)
	case *ast.ConditionalNode:
		return referencesOrderProperty(n.Cond, property, aliases) ||
			referencesOrderProperty(n.Exp1, property, aliases) ||
			referencesOrderProperty(n.Exp2, property, aliases)
	case *ast.ArrayNode:
		return referencesOrderPropertyList(n.Nodes, property, aliases)
	case *ast.MapNode:
		return referencesOrderPropertyList(n.Pairs, property, aliases)
	case *ast.PairNode:
		return referencesOrderProperty(n.Key, property, aliases) ||
			referencesOrderProperty(n.Value, property, aliases)
	default:
		return false
	}
}

func referencesOrderPropertyList(nodes []ast.Node, property string, aliases map[string]struct{}) bool {
	for _, child := range nodes {
		if referencesOrderProperty(child, property, aliases) {
			return true
		}
	}
	return false
}

func inheritOrderAliases(aliases map[string]struct{}) map[string]struct{} {
	inner := make(map[string]struct{}, len(aliases))
	for name := range aliases {
		inner[name] = struct{}{}
	}
	return inner
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
