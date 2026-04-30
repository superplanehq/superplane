package expressionvalidation

import (
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// ParseReferencedNodes returns the names referenced via $['Name']
// inside the given expression body, in source order.
func ParseReferencedNodes(expression string) ([]string, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}

	collector := &nodeReferenceCollector{seen: make(map[string]struct{})}
	ast.Walk(&tree.Node, collector)
	return collector.identifiers, nil
}

type nodeReferenceCollector struct {
	identifiers []string
	seen        map[string]struct{}
}

func (c *nodeReferenceCollector) Visit(node *ast.Node) {
	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	root, ok := member.Node.(*ast.IdentifierNode)
	if !ok || root.Value != "$" {
		return
	}

	switch property := member.Property.(type) {
	case *ast.StringNode:
		c.add(property.Value)
	case *ast.IdentifierNode:
		c.add(property.Value)
	}
}

func (c *nodeReferenceCollector) add(value string) {
	if value == "" {
		return
	}
	if _, ok := c.seen[value]; ok {
		return
	}
	c.seen[value] = struct{}{}
	c.identifiers = append(c.identifiers, value)
}
