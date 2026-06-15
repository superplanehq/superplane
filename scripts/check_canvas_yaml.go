package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/superplanehq/superplane/pkg/lint/canvasyaml"
)

func main() {
	root := repoRoot()
	patterns := []string{
		filepath.Join(root, "pkg/cli/commands/apps/canvas/templates/*.yaml"),
	}

	var paths []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "canvas yaml lint failed: %v\n", err)
			os.Exit(1)
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "canvas yaml lint found no files to scan")
		os.Exit(1)
	}

	var failed bool
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "canvas yaml lint failed to read %s: %v\n", path, err)
			os.Exit(1)
		}

		issues, err := canvasyaml.LintConfigurationFieldNames(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", relPath(root, path), err)
			failed = true
			continue
		}

		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "%s: %s\n", relPath(root, path), issue.String())
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}

	return filepath.ToSlash(rel)
}
