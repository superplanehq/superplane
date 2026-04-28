package expressionvalidation

import (
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/superplanehq/superplane/pkg/exprruntime"
)

// ValidateExpression checks a {{ ... }} placeholder body for syntax errors,
// unknown node references, function arity mistakes, and unresolved identifiers.
// Returns nil for a valid expression.
func ValidateExpression(raw string, knownNodeNames map[string]struct{}) error {
	if err := validateExpressionAST(raw, knownNodeNames); err != nil {
		return err
	}

	return compileWithStubEnv(strings.TrimSpace(raw), knownNodeNames)
}

// ValidateBareExpression checks the same expression body but skips the strict
// compile-time identifier resolution. Use it for FieldTypeExpression values
// without {{ }} framing: at runtime those flow through as literal strings
// unless a component explicitly evaluates them, so identifiers like "default"
// or "ok" are valid literal payloads, while obvious syntax errors and unknown
// node references still need to be caught.
func ValidateBareExpression(raw string, knownNodeNames map[string]struct{}) error {
	return validateExpressionAST(raw, knownNodeNames)
}

func validateExpressionAST(raw string, knownNodeNames map[string]struct{}) error {
	body := strings.TrimSpace(raw)
	if body == "" {
		return fmt.Errorf("expression body is empty")
	}

	tree, err := parser.Parse(body)
	if err != nil {
		return fmt.Errorf("syntax error: %s", firstLine(err.Error()))
	}

	if err := checkNodeReferences(&tree.Node, knownNodeNames); err != nil {
		return err
	}

	return checkFunctionCalls(&tree.Node)
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx != -1 {
		return s[:idx]
	}
	return s
}

func checkNodeReferences(node *ast.Node, knownNodeNames map[string]struct{}) error {
	refs, err := collectNodeReferences(node)
	if err != nil {
		return err
	}
	for _, name := range refs {
		if _, ok := knownNodeNames[name]; !ok {
			return fmt.Errorf("unknown node reference '%s'", name)
		}
	}
	return nil
}

func collectNodeReferences(node *ast.Node) ([]string, error) {
	collector := &nodeReferenceCollector{seen: make(map[string]struct{})}
	ast.Walk(node, collector)
	return collector.identifiers, nil
}

type callChecker struct {
	err error
}

func (c *callChecker) Visit(node *ast.Node) {
	if c.err != nil {
		return
	}
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}

	switch callee := call.Callee.(type) {
	case *ast.IdentifierNode:
		c.err = checkTopLevelCall(callee.Value, call.Arguments)
	case *ast.MemberNode:
		root, rootOK := callee.Node.(*ast.IdentifierNode)
		prop, propOK := callee.Property.(*ast.StringNode)
		if !rootOK || !propOK {
			return
		}
		if root.Value == "memory" {
			c.err = checkMemoryCall(prop.Value, call.Arguments)
		}
	}
}

func checkFunctionCalls(node *ast.Node) error {
	checker := &callChecker{}
	ast.Walk(node, checker)
	return checker.err
}

func checkTopLevelCall(name string, args []ast.Node) error {
	switch name {
	case "root":
		if len(args) != 0 {
			return fmt.Errorf("root() takes no arguments, got %d", len(args))
		}
	case "previous":
		if len(args) > 1 {
			return fmt.Errorf("previous() accepts zero or one argument, got %d", len(args))
		}
		if len(args) == 1 {
			switch arg := args[0].(type) {
			case *ast.IntegerNode:
				if arg.Value < 1 {
					return fmt.Errorf("previous() depth must be >= 1, got %d", arg.Value)
				}
			case *ast.StringNode, *ast.FloatNode, *ast.BoolNode, *ast.NilNode:
				return fmt.Errorf("previous() depth must be an integer literal")
			}
		}
	}
	return nil
}

func checkMemoryCall(method string, args []ast.Node) error {
	switch method {
	case "find", "findFirst":
		if len(args) != 2 {
			return fmt.Errorf("memory.%s() requires a namespace and matches, got %d arguments", method, len(args))
		}
	}
	return nil
}

func compileWithStubEnv(body string, knownNodeNames map[string]struct{}) error {
	dollar := make(map[string]any, len(knownNodeNames))
	for name := range knownNodeNames {
		dollar[name] = map[string]any{}
	}

	memoryStub := func(params ...any) (any, error) { return nil, nil }
	env := map[string]any{
		"$":      dollar,
		"memory": map[string]any{"find": memoryStub, "findFirst": memoryStub},
		"config": map[string]any{},
	}

	opts := []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
		exprruntime.DateFunctionOption(),
		expr.Function("root", func(params ...any) (any, error) { return nil, nil }),
		expr.Function("previous", func(params ...any) (any, error) { return nil, nil }),
	}

	if _, err := expr.Compile(body, opts...); err != nil {
		return fmt.Errorf("compile error: %s", firstLine(err.Error()))
	}
	return nil
}
