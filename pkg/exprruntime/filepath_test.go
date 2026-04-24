package exprruntime

import (
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCommits mirrors the fixture used in the frontend exprEvaluator.spec.ts tests.
var testCommits = []any{
	map[string]any{
		"added":    []any{"pkg/integrations/github/on_push.go"},
		"modified": []any{},
		"removed":  []any{},
	},
	map[string]any{
		"added":    []any{},
		"modified": []any{"web_src/src/App.tsx"},
		"removed":  []any{"docs/old.md"},
	},
}

func evalFilePathMatches(t *testing.T, expression string, env map[string]any) any {
	t.Helper()
	program, err := expr.Compile(expression, expr.Env(env), expr.AsAny(), FilePathMatchesFunctionOption())
	require.NoError(t, err)
	result, err := expr.Run(program, env)
	require.NoError(t, err)
	return result
}

func Test__FilePathMatches(t *testing.T) {
	env := map[string]any{"commits": testCommits}

	t.Run("returns true when a modified file matches the pattern", func(t *testing.T) {
		assert.Equal(t, true, evalFilePathMatches(t, `filePathMatches(commits, "web_src/**")`, env))
	})

	t.Run("returns true when an added file matches the pattern", func(t *testing.T) {
		assert.Equal(t, true, evalFilePathMatches(t, `filePathMatches(commits, "pkg/**")`, env))
	})

	t.Run("returns true when a removed file matches the pattern", func(t *testing.T) {
		assert.Equal(t, true, evalFilePathMatches(t, `filePathMatches(commits, "docs/**")`, env))
	})

	t.Run("returns false when no file matches the pattern", func(t *testing.T) {
		assert.Equal(t, false, evalFilePathMatches(t, `filePathMatches(commits, "migrations/**")`, env))
	})

	t.Run("single wildcard does not cross path segments", func(t *testing.T) {
		// pkg/integrations/github/on_push.go should NOT match pkg/integrations/*
		assert.Equal(t, false, evalFilePathMatches(t, `filePathMatches(commits, "pkg/integrations/*")`, env))
		// but SHOULD match pkg/integrations/github/*
		assert.Equal(t, true, evalFilePathMatches(t, `filePathMatches(commits, "pkg/integrations/github/*")`, env))
	})

	t.Run("returns false for empty commits", func(t *testing.T) {
		emptyEnv := map[string]any{"commits": []any{}}
		assert.Equal(t, false, evalFilePathMatches(t, `filePathMatches(commits, "pkg/**")`, emptyEnv))
	})

	t.Run("returns false for nil commits", func(t *testing.T) {
		nilEnv := map[string]any{"commits": nil}
		assert.Equal(t, false, evalFilePathMatches(t, `filePathMatches(commits, "pkg/**")`, nilEnv))
	})

	t.Run("supports exact match pattern", func(t *testing.T) {
		assert.Equal(t, true, evalFilePathMatches(t, `filePathMatches(commits, "docs/old.md")`, env))
		assert.Equal(t, false, evalFilePathMatches(t, `filePathMatches(commits, "docs/new.md")`, env))
	})
}

func Test__GlobToRegex(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"pkg/**", "pkg/foo/bar.go", true},
		{"pkg/**", "pkg/foo", true},
		{"pkg/*", "pkg/foo", true},
		{"pkg/*", "pkg/foo/bar.go", false},
		{"*.go", "main.go", true},
		{"*.go", "pkg/main.go", false},
		{"docs/old.md", "docs/old.md", true},
		{"docs/old.md", "docs/new.md", false},
		{"**", "anything/at/all.go", true},
		// dots in patterns are treated as literals, not regex wildcards
		{"file.go", "fileXgo", false},
		// ** matches zero intermediate directories
		{"pkg/**/foo.go", "pkg/foo.go", true},
		{"pkg/**/foo.go", "pkg/a/foo.go", true},
		{"pkg/**/foo.go", "pkg/a/b/foo.go", true},
		{"**/foo.go", "foo.go", true},
		{"**/foo.go", "a/foo.go", true},
		{"**/foo.go", "a/b/foo.go", true},
	}

	for _, tc := range cases {
		t.Run(tc.pattern+"->"+tc.path, func(t *testing.T) {
			re, err := GlobToRegex(tc.pattern)
			require.NoError(t, err)
			assert.Equal(t, tc.matches, re.MatchString(tc.path), "pattern=%q path=%q", tc.pattern, tc.path)
		})
	}
}
