//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"time"

	expr "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
)

const (
	FilterTypeData   = "data"
	FilterTypeHeader = "header"
)

type headerVisitor struct{}

// Visit implements the visitor pattern for header variables.
// Update header map keys to be case insensitive.
func (v *headerVisitor) Visit(node *ast.Node) {
	if memberNode, ok := (*node).(*ast.MemberNode); ok {
		memberName := strings.ToLower(memberNode.Node.String())
		if stringNode, ok := memberNode.Property.(*ast.StringNode); ok {
			stringNode.Value = strings.ToLower(stringNode.Value)
		}

		if memberName == "headers" {
			ast.Patch(node, &ast.MemberNode{
				Node:     &ast.IdentifierNode{Value: memberName},
				Property: memberNode.Property,
				Optional: false,
				Method:   false,
			})
		}
	}
}

// CompileBooleanExpression compiles a boolean expression.
//
// variables: the variables to be used in the expression.
// expression: the expression to be compiled.
// filterType: the type of the filter.
func CompileBooleanExpression(variables map[string]any, expression string, filterType string) (*vm.Program, error) {
	options := []expr.Option{
		expr.Env(variables),
		expr.AsBool(),
		expr.Timezone(time.UTC.String()),
	}

	if filterType == FilterTypeHeader {
		options = append(options, expr.Patch(&headerVisitor{}))
	}

	return expr.Compile(expression, options...)
}

// validateBooleanExpression is the exported function for JavaScript
func validateBooleanExpression(this js.Value, args []js.Value) interface{} {
	if len(args) != 3 {
		return map[string]interface{}{
			"error": "Expected 3 arguments: expression, variables, filterType",
		}
	}

	expression := args[0].String()
	variablesJS := args[1]
	filterType := args[2].String()

	// Convert JavaScript variables object to Go map
	variables := make(map[string]interface{})

	// Get all property names from the JavaScript object
	propertyNames := js.Global().Get("Object").Call("keys", variablesJS)
	length := propertyNames.Get("length").Int()

	for i := 0; i < length; i++ {
		key := propertyNames.Index(i).String()
		value := variablesJS.Get(key)
		variables[key] = jsValueToInterface(value)
	}

	// Compile the expression
	program, err := CompileBooleanExpression(variables, expression, filterType)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Test run the expression to validate it
	_, err = expr.Run(program, variables)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"valid": true,
	}
}

// jsValueToInterface converts a JavaScript value to a Go interface{}
func jsValueToInterface(value js.Value) interface{} {
	switch value.Type() {
	case js.TypeBoolean:
		return value.Bool()
	case js.TypeNumber:
		return value.Float()
	case js.TypeString:
		return value.String()
	case js.TypeObject:
		if value.Get("constructor").Get("name").String() == "Array" {
			length := value.Get("length").Int()
			arr := make([]interface{}, length)
			for i := 0; i < length; i++ {
				arr[i] = jsValueToInterface(value.Index(i))
			}
			return arr
		} else {
			obj := make(map[string]interface{})
			propertyNames := js.Global().Get("Object").Call("keys", value)
			length := propertyNames.Get("length").Int()
			for i := 0; i < length; i++ {
				key := propertyNames.Index(i).String()
				obj[key] = jsValueToInterface(value.Get(key))
			}
			return obj
		}
	case js.TypeNull, js.TypeUndefined:
		return nil
	default:
		return value.String()
	}
}

func main() {
	// Register the function to be callable from JavaScript
	js.Global().Set("validateBooleanExpression", js.FuncOf(validateBooleanExpression))

	// Keep the program running
	select {}
}
