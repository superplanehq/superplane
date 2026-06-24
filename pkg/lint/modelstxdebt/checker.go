package modelstxdebt

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

const DefaultRootDir = "pkg/models"

// Guidance explains the preferred database access pattern when this debt cap is exceeded.
const Guidance = `We are moving away from database.Conn() inside pkg/models and from the FindX / FindXInTransaction dual API.

Why:
- Calling database.Conn() inside model code breaks transaction isolation when the caller already holds a tx
- Conn wrappers plus *InTransaction methods duplicate API surface without adding behavior

Use an explicit *gorm.DB parameter instead:

  func FindCanvas(tx *gorm.DB, orgID, id uuid.UUID) (*Canvas, error) {
      var canvas Canvas
      err := tx.Where("organization_id = ? AND id = ?", orgID, id).First(&canvas).Error
      if err != nil {
          return nil, err
      }
      return &canvas, nil
  }

  // Handler (no surrounding transaction):
  canvas, err := models.FindCanvas(database.DB(ctx), orgID, canvasID)

  // Inside an existing transaction:
  err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
      canvas, err := models.FindCanvas(tx, orgID, canvasID)
      return err
  })

See AGENTS.md "Database Transaction Guidelines".`

type Location struct {
	File string
	Line int
	Name string
	key  string
}

// Key returns a stable identifier for baseline comparison. Line numbers are
// intentionally omitted so edits above a call site do not churn the baseline.
func (l Location) Key() string {
	if l.key != "" {
		return l.key
	}

	return fmt.Sprintf("%s:%s", l.File, l.Name)
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d %s", l.File, l.Line, l.Name)
}

type Result struct {
	InTransactionDefinitions []Location
	DatabaseConnCalls        []Location
}

func (r Result) InTransactionDefinitionCount() int {
	return len(r.InTransactionDefinitions)
}

func (r Result) DatabaseConnCallCount() int {
	return len(r.DatabaseConnCalls)
}

// Scan walks rootDir for non-test Go files and counts *InTransaction definitions
// and database.Conn() call sites.
func Scan(rootDir string) (Result, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return Result{}, fmt.Errorf("stat scan root %q: %w", rootDir, err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("scan root %q is not a directory", rootDir)
	}

	var result Result
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

		fileResult, parseErr := scanFile(fset, path)
		if parseErr != nil {
			return parseErr
		}

		result.InTransactionDefinitions = append(result.InTransactionDefinitions, fileResult.InTransactionDefinitions...)
		result.DatabaseConnCalls = append(result.DatabaseConnCalls, fileResult.DatabaseConnCalls...)
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	sortLocations(result.InTransactionDefinitions)
	sortLocations(result.DatabaseConnCalls)

	return result, nil
}

func scanFile(fset *token.FileSet, path string) (Result, error) {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", path, err)
	}

	databaseIdent := databaseImportIdent(file)
	var result Result

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}

		if strings.HasSuffix(fn.Name.Name, "InTransaction") {
			result.InTransactionDefinitions = append(result.InTransactionDefinitions, Location{
				File: path,
				Line: fset.Position(fn.Name.Pos()).Line,
				Name: fn.Name.Name,
				key:  fmt.Sprintf("%s:%s", path, fn.Name.Name),
			})
		}

		if fn.Body == nil {
			continue
		}

		connOrdinal := 0
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !isDatabaseConnCall(call, databaseIdent) {
				return true
			}

			connOrdinal++
			result.DatabaseConnCalls = append(result.DatabaseConnCalls, Location{
				File: path,
				Line: fset.Position(call.Pos()).Line,
				Name: "database.Conn()",
				key:  fmt.Sprintf("%s:%s:database.Conn()#%d", path, fn.Name.Name, connOrdinal),
			})

			return true
		})
	}

	return result, nil
}

func databaseImportIdent(file *ast.File) string {
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil || importPath != "github.com/superplanehq/superplane/pkg/database" {
			continue
		}

		if importSpec.Name != nil {
			return importSpec.Name.Name
		}

		return "database"
	}

	return "database"
}

func isDatabaseConnCall(call *ast.CallExpr, databaseIdent string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil || selector.Sel.Name != "Conn" {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == databaseIdent
}

func sortLocations(locations []Location) {
	slices.SortFunc(locations, func(a, b Location) int {
		if cmp := strings.Compare(a.File, b.File); cmp != 0 {
			return cmp
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return strings.Compare(a.Name, b.Name)
	})
}
