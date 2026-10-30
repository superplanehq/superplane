package runcodeagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__parseGitHubIssueOrigin(t *testing.T) {
	t.Run("github issue URL", func(t *testing.T) {
		ref, ok := parseGitHubIssueOrigin("https://github.com/acme/widgets/issues/42")
		require.True(t, ok)
		assert.Equal(t, IssueRef{Repository: "acme/widgets", Number: 42, host: githubcommon.DotCom()}, ref)
	})

	t.Run("github issue URL with trailing fragment", func(t *testing.T) {
		ref, ok := parseGitHubIssueOrigin("https://github.com/acme/widgets/issues/42#issuecomment-1")
		require.True(t, ok)
		assert.Equal(t, IssueRef{Repository: "acme/widgets", Number: 42, host: githubcommon.DotCom()}, ref)
	})

	t.Run("enterprise issue URL keeps the host", func(t *testing.T) {
		ref, ok := parseGitHubIssueOrigin("https://acme.ghe.com/acme/widgets/issues/42")
		require.True(t, ok)
		host, err := githubcommon.ParseHost("acme.ghe.com")
		require.NoError(t, err)
		assert.Equal(t, IssueRef{Repository: "acme/widgets", Number: 42, host: host}, ref)
	})

	t.Run("github pull request URL is not an issue", func(t *testing.T) {
		_, ok := parseGitHubIssueOrigin("https://github.com/acme/widgets/pull/42")
		assert.False(t, ok)
	})

	t.Run("non-github origin", func(t *testing.T) {
		_, ok := parseGitHubIssueOrigin("https://linear.app/acme/issue/ENG-123")
		assert.False(t, ok)
	})

	t.Run("malformed URL", func(t *testing.T) {
		_, ok := parseGitHubIssueOrigin("not a url")
		assert.False(t, ok)
	})

	t.Run("empty URL", func(t *testing.T) {
		_, ok := parseGitHubIssueOrigin("")
		assert.False(t, ok)
	})
}

func Test__issueBacklinkReference(t *testing.T) {
	issue := IssueRef{Repository: "acme/widgets", Number: 42}
	dotcom := githubcommon.DotCom()

	t.Run("same repository (owner/repo form)", func(t *testing.T) {
		assert.Equal(t, "#42", issueBacklinkReference(issue, "acme/widgets", dotcom))
	})

	t.Run("same repository (clone URL form)", func(t *testing.T) {
		assert.Equal(t, "#42", issueBacklinkReference(issue, "https://github.com/acme/widgets.git", dotcom))
	})

	t.Run("same repository, case-insensitive", func(t *testing.T) {
		assert.Equal(t, "#42", issueBacklinkReference(issue, "Acme/Widgets", dotcom))
	})

	t.Run("cross-repository", func(t *testing.T) {
		assert.Equal(t, "acme/widgets#42", issueBacklinkReference(issue, "acme/other-repo", dotcom))
	})

	t.Run("unresolved template expression falls back to cross-repo form", func(t *testing.T) {
		assert.Equal(t, "acme/widgets#42", issueBacklinkReference(issue, "{{ event.repo }}", dotcom))
	})

	t.Run("enterprise origin on a public GitHub target is omitted", func(t *testing.T) {
		host, err := githubcommon.ParseHost("acme.ghe.com")
		require.NoError(t, err)
		enterprise := IssueRef{Repository: "acme/widgets", Number: 42, host: host}
		assert.Equal(t, "", issueBacklinkReference(enterprise, "acme/widgets", dotcom))
		assert.Equal(t, "#42", issueBacklinkReference(enterprise, "https://acme.ghe.com/acme/widgets.git", host))
	})
}

func Test__resolveResolvesIssue(t *testing.T) {
	t.Run("no expression context", func(t *testing.T) {
		assert.Nil(t, resolveResolvesIssue(core.ExecutionContext{}))
	})

	t.Run("no origin", func(t *testing.T) {
		ctx := core.ExecutionContext{Expressions: &contexts.ExpressionContext{Output: nil}}
		assert.Nil(t, resolveResolvesIssue(ctx))
	})

	t.Run("github issue origin", func(t *testing.T) {
		ctx := core.ExecutionContext{Expressions: &contexts.ExpressionContext{
			Output: map[string]any{"url": "https://github.com/acme/widgets/issues/7", "label": "acme/widgets#7"},
		}}
		ref := resolveResolvesIssue(ctx)
		require.NotNil(t, ref)
		assert.Equal(t, IssueRef{Repository: "acme/widgets", Number: 7, host: githubcommon.DotCom()}, *ref)
	})

	t.Run("non-github origin", func(t *testing.T) {
		ctx := core.ExecutionContext{Expressions: &contexts.ExpressionContext{
			Output: map[string]any{"url": "https://linear.app/acme/issue/ENG-123", "label": "ENG-123"},
		}}
		assert.Nil(t, resolveResolvesIssue(ctx))
	})

	t.Run("github pull request origin", func(t *testing.T) {
		ctx := core.ExecutionContext{Expressions: &contexts.ExpressionContext{
			Output: map[string]any{"url": "https://github.com/acme/widgets/pull/7", "label": "acme/widgets#7"},
		}}
		assert.Nil(t, resolveResolvesIssue(ctx))
	})

	t.Run("expression error", func(t *testing.T) {
		ctx := core.ExecutionContext{Expressions: &contexts.ExpressionContext{
			Output: map[string]any{"url": "https://github.com/acme/widgets/issues/7"},
			Error:  assert.AnError,
		}}
		assert.Nil(t, resolveResolvesIssue(ctx))
	})
}
