package factories

import (
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrFeedbackActivityDescriptionExpression_MissingCommentOrReview(t *testing.T) {
	source := templateExpressionSource(t, prFeedbackActivityDescriptionExpression())

	cases := []struct {
		name string
		data map[string]any
		want string
	}{
		{
			name: "review without comment",
			data: map[string]any{"review": map[string]any{"body": "LGTM"}},
			want: "LGTM",
		},
		{
			name: "comment without review",
			data: map[string]any{"comment": map[string]any{"body": "please add tests"}},
			want: "please add tests",
		},
		{
			name: "inline review comments when review body is empty",
			data: map[string]any{
				"review": map[string]any{"body": nil, "state": "commented"},
				"review_comments": []any{
					map[string]any{"body": "@superplaneagent upgrade to Go 1.26"},
				},
			},
			want: "@superplaneagent upgrade to Go 1.26",
		},
		{
			name: "neither comment nor review",
			data: map[string]any{},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, evalRootDataExpression(t, source, tc.data))
		})
	}
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
