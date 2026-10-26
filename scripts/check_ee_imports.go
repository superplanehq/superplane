//go:build ignore

// Check that production core packages access Enterprise code only through server.
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const enterpriseImport = "github.com/superplanehq/superplane/ee"

func main() {
	violations, err := checkEnterpriseImports()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(violations) != 0 {
		fmt.Fprintln(os.Stderr, "Production core packages can import EE only from pkg/server:")
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, violation)
		}
		os.Exit(1)
	}
	fmt.Println("Enterprise import boundary passed.")
}

// Parse imports without evaluating build tags so alternate builds have the
// same boundary. Test files and generated SDKs do not define this boundary.
func checkEnterpriseImports() ([]string, error) {
	var violations []string
	for _, root := range []string{"pkg", "cmd", "scripts"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			normalized := filepath.ToSlash(path)
			if entry.IsDir() {
				if normalized == "pkg/protos" || normalized == "pkg/openapi_client" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || filepath.ToSlash(filepath.Dir(path)) == "pkg/server" {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			for _, dependency := range file.Imports {
				imported, err := strconv.Unquote(dependency.Path.Value)
				if err != nil {
					return fmt.Errorf("read import in %s: %w", path, err)
				}
				if imported == enterpriseImport || strings.HasPrefix(imported, enterpriseImport+"/") {
					violations = append(violations, fmt.Sprintf("%s imports %s", normalized, imported))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return violations, nil
}
