package factories

import (
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrFeedbackCommentActivityDescriptionExpression(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackCommentActivityDescriptionExpression())

	data := map[string]any{"comment": map[string]any{"body": "please add tests"}}
	assert.Equal(t, "please add tests", evalRootDataExpression(t, source, data))
}

func TestPrFeedbackReviewActivityDescriptionExpression(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackReviewActivityDescriptionExpression())

	cases := []struct {
		name string
		data map[string]any
		want string
	}{
		{
			name: "review body without inline comments",
			data: map[string]any{"review": map[string]any{"body": "LGTM"}},
			want: "LGTM",
		},
		{
			name: "inline review comments when review body is empty",
			data: map[string]any{
				"review": map[string]any{"body": nil, "state": "commented"},
				"review_comments": []any{
					map[string]any{
						"body":     "@superplaneagent upgrade to Go 1.26",
						"path":     "go.mod",
						"html_url": "https://github.com/acme/app/pull/42#discussion_r1",
					},
				},
			},
			want: "[go.mod](https://github.com/acme/app/pull/42#discussion_r1)\n" +
				"@superplaneagent upgrade to Go 1.26",
		},
		{
			name: "review body and inline comments",
			data: map[string]any{
				"review": map[string]any{"body": "Review summary"},
				"review_comments": []any{
					map[string]any{
						"body":     "First comment",
						"path":     "docs/API.md",
						"html_url": "https://github.com/acme/app/pull/42#discussion_r1",
					},
					map[string]any{
						"body":     "Second comment",
						"path":     "pkg/models/decks.go",
						"html_url": "https://github.com/acme/app/pull/42#discussion_r2",
					},
				},
			},
			want: "Review summary\n\n---\n\n" +
				"[docs/API.md](https://github.com/acme/app/pull/42#discussion_r1)\nFirst comment\n\n---\n\n" +
				"[pkg/models/decks.go](https://github.com/acme/app/pull/42#discussion_r2)\nSecond comment",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, evalRootDataExpression(t, source, tc.data))
		})
	}
}

func TestPrFeedbackCommentActivityTitleExpression(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackCommentActivityTitleExpression())
	data := map[string]any{
		"comment": map[string]any{
			"body":     "please add tests",
			"html_url": "https://github.com/acme/app/pull/42#issuecomment-1",
			"user": map[string]any{
				"login":    "lucaspin",
				"html_url": "https://github.com/lucaspin",
			},
		},
	}

	assert.Equal(
		t,
		"[@lucaspin](https://github.com/lucaspin) left a [comment](https://github.com/acme/app/pull/42#issuecomment-1)",
		evalRootDataExpression(t, source, data),
	)
}

func TestPrFeedbackReplyActivityTitleExpression(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackReplyActivityTitleExpression())
	data := map[string]any{
		"comment": map[string]any{
			"body":     "please add tests",
			"html_url": "https://github.com/acme/app/pull/42#discussion_r1",
			"path":     "pkg/core/integration.go",
			"user": map[string]any{
				"login":    "lucaspin",
				"html_url": "https://github.com/lucaspin",
			},
		},
	}

	assert.Equal(
		t,
		"[@lucaspin](https://github.com/lucaspin) left a [comment](https://github.com/acme/app/pull/42#discussion_r1) in `pkg/core/integration.go`",
		evalRootDataExpression(t, source, data),
	)
}

func TestPrFeedbackReviewActivityTitleExpression(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackReviewActivityTitleExpression())
	data := map[string]any{
		"review": map[string]any{
			"body":     "review summary",
			"html_url": "https://github.com/acme/app/pull/42#pullrequestreview-1",
			"user": map[string]any{
				"login":    "lucaspin",
				"html_url": "https://github.com/lucaspin",
			},
		},
		"review_comments": []any{
			map[string]any{"body": "first"},
			map[string]any{"body": "second"},
		},
	}

	assert.Equal(
		t,
		"[@lucaspin](https://github.com/lucaspin) left a [review](https://github.com/acme/app/pull/42#pullrequestreview-1) - addressing",
		evalRootDataExpression(t, source, data),
	)
}

func TestPrFeedbackPRNumberExpression_IssueCommentPayload(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackPRNumberExpression())

	got := evalRootDataExpression(t, source, map[string]any{
		"issue": map[string]any{"number": 42},
	})
	assert.Equal(t, 42, got)
}

func templateExpressionSource(t *testing.T, wrapped string) string {
	t.Helper()

	source := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(wrapped, "{{"), "}}"))
	require.NotEmpty(t, source)
	return source
}

func evalRootDataExpression(t *testing.T, source string, data map[string]any) any {
	t.Helper()

	payload := map[string]any{"data": data}
	program, err := expr.Compile(source, expr.AsAny(), expr.Function("root", func(params ...any) (any, error) {
		return payload, nil
	}))
	require.NoError(t, err)

	got, err := expr.Run(program, map[string]any{})
	require.NoError(t, err)
	return got
}
