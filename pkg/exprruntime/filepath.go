package exprruntime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

// FilePathMatchesFunctionOption registers the filePathMatches() helper with the expr engine.
//
// Usage: filePathMatches(commits, pattern)
//
// Returns true if any file (added, modified, or removed) across the given commits matches
// the glob pattern. Supports:
//   - **  matches any sequence of path components (including across directory separators)
//   - *   matches any sequence of characters within a single path segment (no /)
//
// Example:
//
//	filePathMatches(root().data.commits, "pkg/**")
func FilePathMatchesFunctionOption() expr.Option {
	return expr.Function("filePathMatches", func(params ...any) (any, error) {
		if len(params) != 2 {
			return nil, fmt.Errorf("filePathMatches() expects 2 arguments: commits and pattern")
		}

		pattern, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("filePathMatches() pattern must be a string, got %T", params[1])
		}

		re, err := globToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("filePathMatches() invalid pattern %q: %w", pattern, err)
		}

		if params[0] == nil {
			return false, nil
		}

		commits, ok := params[0].([]any)
		if !ok {
			return nil, fmt.Errorf("filePathMatches() commits must be an array, got %T", params[0])
		}

		for _, c := range commits {
			commit, ok := c.(map[string]any)
			if !ok {
				continue
			}

			for _, key := range []string{"added", "modified", "removed"} {
				files, ok := commit[key].([]any)
				if !ok {
					continue
				}

				for _, f := range files {
					path, ok := f.(string)
					if !ok {
						continue
					}

					if re.MatchString(path) {
						return true, nil
					}
				}
			}
		}

		return false, nil
	})
}

// globToRegex converts a glob pattern to a compiled regexp.
// Supported syntax:
//   - ** matches any sequence of path components (including zero)
//   - *  matches any sequence of characters within a single path segment (no /)
//   - All other regexp metacharacters are escaped
func globToRegex(pattern string) (*regexp.Regexp, error) {
	var buf strings.Builder
	buf.WriteString("^")

	i := 0
	for i < len(pattern) {
		if pattern[i] == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				buf.WriteString(".*")
				i += 2
			} else {
				buf.WriteString("[^/]*")
				i++
			}
		} else {
			if isRegexMeta(pattern[i]) {
				buf.WriteByte('\\')
			}
			buf.WriteByte(pattern[i])
			i++
		}
	}

	buf.WriteString("$")
	return regexp.Compile(buf.String())
}

// isRegexMeta reports whether b is a regexp metacharacter.
// * is excluded because it is handled as a glob wildcard.
func isRegexMeta(b byte) bool {
	switch b {
	case '.', '+', '?', '^', '$', '{', '}', '(', ')', '|', '[', ']', '\\':
		return true
	}
	return false
}
