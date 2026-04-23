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

		re, err := GlobToRegex(pattern)
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

// GlobToRegex converts a glob pattern to a compiled regexp.
// Supported syntax (identical to GitHub Actions path filters):
//   - **  matches any sequence of path components including zero (crosses /)
//   - *   matches any sequence of characters within a single path segment (no /)
//   - All other regexp metacharacters are escaped
//
// Zero-directory semantics:
//   - /**/  matches either / alone or /anything/ (zero or more intermediate dirs)
//   - **/   at pattern start matches an optional dir prefix (e.g. **/foo.go matches foo.go)
func GlobToRegex(pattern string) (*regexp.Regexp, error) {
	var buf strings.Builder
	buf.WriteString("^")

	// Replace /**/  with a placeholder so the per-character loop can emit the
	// correct alternation without having already consumed the surrounding slashes.
	pattern = strings.ReplaceAll(pattern, "/**/", "\x00")

	i := 0
	for i < len(pattern) {
		switch {
		case pattern[i] == '\x00':
			// Was /**/ — zero or more intermediate path components.
			// Matches either a single / (zero intermediate dirs) or /…/ (one or more).
			buf.WriteString("(/|/.+/)")
			i++

		case pattern[i] == '*' && i+1 < len(pattern) && pattern[i+1] == '*':
			if i == 0 && i+2 < len(pattern) && pattern[i+2] == '/' {
				// **/ at the very start — optional dir prefix with trailing slash.
				buf.WriteString("(.+/)?")
				i += 3
			} else {
				// ** at end or after a non-slash — match anything.
				buf.WriteString(".*")
				i += 2
			}

		case pattern[i] == '*':
			buf.WriteString("[^/]*")
			i++

		default:
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
