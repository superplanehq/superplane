package pathfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__TrimNonEmptyStrings(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, TrimNonEmptyStrings([]string{"a", "", "  ", "b"}))
	assert.Nil(t, TrimNonEmptyStrings([]string{"", "  ", ""}))
}

func Test__changedFilesMatchPushPathGlobs(t *testing.T) {
	noLog := func(string, error) {}

	t.Run("empty patterns -> false", func(t *testing.T) {
		assert.False(t, changedFilesMatchPushPathGlobs([]string{}, []string{"a.go"}, noLog))
	})

	t.Run("empty changed files -> false", func(t *testing.T) {
		assert.False(t, changedFilesMatchPushPathGlobs([]string{"**"}, []string{}, noLog))
	})

	t.Run("simple include matches", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"pkg/**"},
			[]string{"pkg/models/x.go"},
			noLog,
		))
	})

	t.Run("does not match vendor/pkg as pkg prefix", func(t *testing.T) {
		assert.False(t, changedFilesMatchPushPathGlobs(
			[]string{"pkg/**"},
			[]string{"vendor/pkg/foo.go"},
			noLog,
		))
	})

	t.Run("exact path", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"go.sum"},
			[]string{"go.mod", "go.sum"},
			noLog,
		))
	})

	t.Run("exclude markdown under billing", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"billing/**", "!billing/**/*.md"},
			[]string{"billing/service/main.go"},
			noLog,
		))
		assert.False(t, changedFilesMatchPushPathGlobs(
			[]string{"billing/**", "!billing/**/*.md"},
			[]string{"billing/README.md"},
			noLog,
		))
	})

	t.Run("exclude only implies ** include", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"!docs/**"},
			[]string{"pkg/main.go"},
			noLog,
		))
		assert.False(t, changedFilesMatchPushPathGlobs(
			[]string{"!docs/**"},
			[]string{"docs/README.md"},
			noLog,
		))
	})

	t.Run("trim and skip blank patterns", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"  pkg/**  ", "", " "},
			[]string{"pkg/x.go"},
			noLog,
		))
	})

	t.Run("leading ** matches file at repo root", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"**/foo.go"},
			[]string{"foo.go"},
			noLog,
		))
	})

	t.Run("zero intermediate directories with ** in middle", func(t *testing.T) {
		assert.True(t, changedFilesMatchPushPathGlobs(
			[]string{"pkg/**/bar.go"},
			[]string{"pkg/bar.go"},
			noLog,
		))
	})

	t.Run("invalid positive glob only -> bypass allows any files", func(t *testing.T) {
		onInv := func(string) {}
		var bypass string
		assert.True(t, EvaluatePushPathGlobFilter(
			[]string{"[["},
			[]string{"any.go"},
			onInv,
			nil,
			func(r string) { bypass = r },
		))
		assert.Contains(t, bypass, "invalid")
	})

	t.Run("invalid positive but valid exclude still restricts", func(t *testing.T) {
		onInv := func(string) {}
		assert.True(t, EvaluatePushPathGlobFilter(
			[]string{"[[", "!pkg/**"},
			[]string{"README.md"},
			onInv,
			nil,
			nil,
		))
		assert.False(t, EvaluatePushPathGlobFilter(
			[]string{"[[", "!pkg/**"},
			[]string{"pkg/foo.go"},
			onInv,
			nil,
			nil,
		))
	})

	t.Run("multiple changed files fire if any path passes filter", func(t *testing.T) {
		assert.True(t, EvaluatePushPathGlobFilter(
			[]string{"billing/**", "!billing/**/*.md"},
			[]string{"billing/README.md", "billing/service/main.go"},
			nil,
			nil,
			nil,
		))
	})

	t.Run("empty pattern slice returns true (vacuous pass)", func(t *testing.T) {
		assert.True(t, EvaluatePushPathGlobFilter(nil, []string{"a.go"}, nil, nil, nil))
	})

	t.Run("exclude only with invalid exclude globs bypasses", func(t *testing.T) {
		onInv := func(string) {}
		var bypass string
		ok := EvaluatePushPathGlobFilter(
			[]string{"![["},
			[]string{"pkg/x.go"},
			onInv,
			nil,
			func(reason string) { bypass = reason },
		)
		assert.True(t, ok)
		assert.NotEmpty(t, bypass)
	})
}
