package grpcactionsstatus

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	DefaultRootDir   = "pkg/grpc/actions"
	StatusImportPath = "google.golang.org/grpc/status"
)

// Guidance explains why handlers should not import google.golang.org/grpc/status.
const Guidance = `pkg/grpc/actions handlers must not use google.golang.org/grpc/status directly.

Use pkg/grpcerrors instead so underlying errors (including context cancellation) reach
the grpc-gateway sanitizer:

  return nil, grpcerrors.Internal(err, "failed to load canvas")
  return nil, grpcerrors.NotFound(err, "canvas not found")

Keep explicit non-Internal gRPC codes as status.Error only until grpcerrors helpers exist,
or return raw errors and map them at the gateway. New handler code should prefer grpcerrors.`

type Violation struct {
	File   string
	Line   int
	Detail string
}

func (v Violation) Key() string {
	return fmt.Sprintf("%s:%d:%s", v.File, v.Line, v.Detail)
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d %s", v.File, v.Line, v.Detail)
}

// Scan walks rootDir for non-test Go files and reports imports or uses of
// google.golang.org/grpc/status.
func Scan(rootDir string) ([]Violation, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("stat scan root %q: %w", rootDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root %q is not a directory", rootDir)
	}

	var violations []Violation
	fset := token.NewFileSet()

	err = filepath.WalkDir(rootDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileViolations, scanErr := scanFile(fset, path)
		if scanErr != nil {
			return scanErr
		}

		violations = append(violations, fileViolations...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(violations, func(a, b Violation) int {
		if cmp := strings.Compare(a.File, b.File); cmp != 0 {
			return cmp
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return strings.Compare(a.Detail, b.Detail)
	})

	return violations, nil
}

func scanFile(fset *token.FileSet, path string) ([]Violation, error) {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	statusIdent := statusImportIdent(file)
	if statusIdent == "" {
		return nil, nil
	}

	var violations []Violation
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil || importPath != StatusImportPath {
			continue
		}

		violations = append(violations, Violation{
			File:   path,
			Line:   fset.Position(importSpec.Pos()).Line,
			Detail: "imports google.golang.org/grpc/status",
		})
		break
	}

	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != statusIdent {
			return true
		}

		violations = append(violations, Violation{
			File:   path,
			Line:   fset.Position(selector.Pos()).Line,
			Detail: fmt.Sprintf("uses status.%s", selector.Sel.Name),
		})
		return true
	})

	return violations, nil
}

func statusImportIdent(file *ast.File) string {
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil || importPath != StatusImportPath {
			continue
		}

		if importSpec.Name != nil {
			return importSpec.Name.Name
		}

		return "status"
	}

	return ""
}
