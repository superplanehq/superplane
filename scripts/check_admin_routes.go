//go:build ignore

// Reject new manual admin routes while existing handlers migrate to protos/admin.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const adminPrefix = "/admin/api"

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if err := checkAdminRoutes(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkAdminRoutes(root string) error {
	baseline, err := os.ReadFile(filepath.Join(root, ".admin-http-routes-baseline.txt"))
	if err != nil {
		return fmt.Errorf("read admin route baseline: %w", err)
	}
	allowed := map[string]bool{}
	for _, line := range strings.Split(string(baseline), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			allowed[line] = true
		}
	}
	current := map[string]bool{}
	var violations []string
	for _, directory := range []string{"pkg", "ee", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if relative == "pkg/protos" || relative == "pkg/openapi_client" {
				return fs.SkipDir
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			positions := token.NewFileSet()
			file, err := parser.ParseFile(positions, path, nil, 0)
			if err != nil {
				return fmt.Errorf("parse %s: %w", relative, err)
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Body == nil {
					continue
				}
				for _, route := range manualAdminRoutes(function) {
					key := relative + " " + route.method + " " + route.path
					current[key] = true
					if !allowed[key] {
						violations = append(violations, fmt.Sprintf("%s:%d: %s %s", relative, positions.Position(route.position).Line, route.method, route.path))
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan admin routes: %w", err)
		}
	}
	if len(violations) > 0 {
		slices.Sort(violations)
		return fmt.Errorf("new manual admin routes:\n  %s\nDefine new admin endpoints in protos/admin/ and register them with the admin gRPC gateway. Do not expand the baseline", strings.Join(violations, "\n  "))
	}
	var obsolete []string
	for key := range allowed {
		if !current[key] {
			obsolete = append(obsolete, key)
		}
	}
	if len(obsolete) > 0 {
		slices.Sort(obsolete)
		return fmt.Errorf("remove migrated routes from .admin-http-routes-baseline.txt:\n  %s", strings.Join(obsolete, "\n  "))
	}
	fmt.Printf("Admin route check passed: %d existing manual routes; new endpoints must use protos/admin/.\n", len(current))
	return nil
}

type adminRoute struct {
	path     string
	method   string
	position token.Pos
}

type routeBindings struct {
	adminRouters map[string]string
	gateways     map[string]bool
}

func manualAdminRoutes(function *ast.FuncDecl) []adminRoute {
	bindings := routeBindings{adminRouters: map[string]string{}, gateways: map[string]bool{}}
	if function.Name.Name == "RegisterAdmin" {
		for _, parameter := range function.Type.Params.List {
			for _, name := range parameter.Names {
				bindings.adminRouters[name.Name] = ""
			}
		}
	}
	var calls []*ast.CallExpr
	ast.Inspect(function.Body, func(node ast.Node) bool {
		// Track the router and gateway aliases used by route registration helpers.
		if assignment, ok := node.(*ast.AssignStmt); ok {
			for index, value := range assignment.Rhs {
				if index >= len(assignment.Lhs) {
					break
				}
				name, ok := assignment.Lhs[index].(*ast.Ident)
				if !ok {
					continue
				}
				if prefix, ok := bindings.adminRouter(value); ok {
					bindings.adminRouters[name.Name] = prefix
				}
				if bindings.gateway(value) {
					bindings.gateways[name.Name] = true
				}
			}
		}
		if call, ok := node.(*ast.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})
	var routes []adminRoute
	for _, call := range calls {
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !slices.Contains([]string{"Handle", "HandleFunc", "Handler", "HandlerFunc"}, selector.Sel.Name) {
			continue
		}
		prefix, admin := bindings.adminRouter(selector.X)
		path := prefix
		handlerIndex := 0
		if selector.Sel.Name == "Handle" || selector.Sel.Name == "HandleFunc" {
			if len(call.Args) != 2 {
				continue
			}
			path = stringLiteral(call.Args[0])
			admin = admin || path == adminPrefix || strings.HasPrefix(path, adminPrefix+"/")
			path = prefix + path
			handlerIndex = 1
		}
		if !admin || len(call.Args) <= handlerIndex {
			continue
		}
		handler := call.Args[handlerIndex]
		if bindings.gateway(handler) || isNotFoundHandler(handler) {
			continue
		}
		if path == "" {
			path = "<dynamic>"
		}
		for _, method := range routeMethods(call, calls) {
			routes = append(routes, adminRoute{path: path, method: method, position: call.Pos()})
		}
	}
	return routes
}

func isNotFoundHandler(expression ast.Expr) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "NotFound" {
		return false
	}
	name, ok := selector.X.(*ast.Ident)
	return ok && name.Name == "http"
}

func (b routeBindings) adminRouter(expression ast.Expr) (string, bool) {
	switch expression := expression.(type) {
	case *ast.Ident:
		prefix, ok := b.adminRouters[expression.Name]
		return prefix, ok
	case *ast.SelectorExpr:
		return "", expression.Sel.Name == "adminRouter"
	case *ast.CallExpr:
		selector, ok := expression.Fun.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		prefix, admin := b.adminRouter(selector.X)
		if (selector.Sel.Name == "PathPrefix" || selector.Sel.Name == "Path") && len(expression.Args) == 1 {
			path := stringLiteral(expression.Args[0])
			if path == adminPrefix || strings.HasPrefix(path, adminPrefix+"/") {
				return strings.TrimPrefix(path, adminPrefix), true
			}
			return prefix + path, admin
		}
		return prefix, admin
	}
	return "", false
}

func (b routeBindings) gateway(expression ast.Expr) bool {
	if name, ok := expression.(*ast.Ident); ok {
		return b.gateways[name.Name]
	}
	if call, ok := expression.(*ast.CallExpr); ok {
		selector, ok := call.Fun.(*ast.SelectorExpr)
		return ok && selector.Sel.Name == "grpcGatewayAccountHandler"
	}
	return false
}

func routeMethods(registration *ast.CallExpr, calls []*ast.CallExpr) []string {
	var methods []string
	for _, call := range calls {
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Methods" || (!inCallChain(selector.X, registration) && !inCallChain(registration, call)) {
			continue
		}
		for _, argument := range call.Args {
			method := stringLiteral(argument)
			if selector, ok := argument.(*ast.SelectorExpr); ok && strings.HasPrefix(selector.Sel.Name, "Method") {
				method = strings.ToUpper(strings.TrimPrefix(selector.Sel.Name, "Method"))
			}
			methods = append(methods, method)
		}
	}
	if len(methods) == 0 {
		return []string{"*"}
	}
	return methods
}

func inCallChain(expression ast.Expr, target *ast.CallExpr) bool {
	if expression == target {
		return true
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && inCallChain(selector.X, target)
}

func stringLiteral(expression ast.Expr) string {
	if literal, ok := expression.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		value, err := strconv.Unquote(literal.Value)
		if err == nil {
			return value
		}
	}
	return "<dynamic>"
}
