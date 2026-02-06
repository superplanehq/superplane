package contexts

import (
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// expressionContainsSecrets reports whether the expression contains a call to
// secrets() and must be deferred to runtime resolution. 
//
// If parsing fails, it falls back to a string check so malformed expressions are still
// deferred and will fail at runtime.
func expressionContainsSecrets(expression string) bool {
	tree, err := parser.Parse(expression)

	if err != nil {
		// If parsing fails, fall back to a string check to ensure malformed expressions are still deferred.
		return strings.Contains(expression, "secrets(")
	}

	collector := &secretsCallCollector{}
	ast.Walk(&tree.Node, collector)
	return collector.found
}

type secretsCallCollector struct {
	found bool
}

func (c *secretsCallCollector) Visit(node *ast.Node) {
	if c.found {
		return
	}

	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}

	if id, ok := call.Callee.(*ast.IdentifierNode); ok && id.Value == "secrets" {
		c.found = true
	}
}
