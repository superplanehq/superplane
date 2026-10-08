package factories

import (
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goyaml "gopkg.in/yaml.v3"
)

func TestPrFeedbackChecksWaitingTitleExpression(t *testing.T) {
	assert.Equal(
		t,
		"Waiting for checks on [3fc0c4c](https://github.com/acme/app/commit/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksWaitingTitleExpression()), checksTitleData()),
	)
}

func TestPrFeedbackChecksWaitingTitleExpression_FallsBackToFullName(t *testing.T) {
	data := checksTitleData()
	delete(data["repository"].(map[string]any), "html_url")
	assert.Equal(
		t,
		"Waiting for checks on [3fc0c4c](https://github.com/acme/app/commit/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksWaitingTitleExpression()), data),
	)
}

func TestPrFeedbackChecksPassedTitleExpression(t *testing.T) {
	assert.Equal(
		t,
		"Checks passed on [3fc0c4c](https://github.com/acme/app/commit/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksPassedTitleExpression()), checksTitleData()),
	)
}

func TestPrFeedbackChecksRepairTitleExpression(t *testing.T) {
	assert.Equal(
		t,
		"Fixing failed checks on [3fc0c4c](https://github.com/acme/app/commit/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksRepairTitleExpression()), checksTitleData()),
	)
}

func bitbucketChecksTitleData() map[string]any {
	return map[string]any{
		"repository": map[string]any{
			"full_name": "acme/widgets",
			"links": map[string]any{
				"html": map[string]any{"href": "https://bitbucket.org/acme/widgets"},
			},
		},
		"pull_request": map[string]any{
			"head": map[string]any{
				"sha": "3fc0c4c0123456789abcdef",
			},
		},
	}
}

func TestPrFeedbackBitbucketTitleExpressions(t *testing.T) {
	assert.Equal(
		t,
		"Waiting for builds on [3fc0c4c](https://bitbucket.org/acme/widgets/commits/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksWaitingTitleExpressionFor("bitbucket")), bitbucketChecksTitleData()),
	)
	assert.Equal(
		t,
		"Builds passed on [3fc0c4c](https://bitbucket.org/acme/widgets/commits/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksPassedTitleExpressionFor("bitbucket")), bitbucketChecksTitleData()),
	)

	fallback := bitbucketChecksTitleData()
	delete(fallback["repository"].(map[string]any), "links")
	assert.Equal(
		t,
		"Fixing failed builds on [3fc0c4c](https://bitbucket.org/acme/widgets/commits/3fc0c4c0123456789abcdef)",
		evalRootDataExpression(t, templateExpressionSource(t, prFeedbackChecksRepairTitleExpressionFor("bitbucket")), fallback),
	)
}

func TestBuildChecksPRFeedbackCanvasBitbucket(t *testing.T) {
	canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository:  "acme/widgets",
		CheckNames:  []string{"build-a"},
		VCSProvider: "bitbucket",
		Agent: &intakeAgent{
			Component: "runnerOpenRouter",
			Model:     "anthropic/claude-sonnet-4-6",
		},
	})

	trigger := findSpecNode(t, canvas, prFeedbackPullRequestTriggerNodeID)
	assert.Equal(t, "bitbucket.onPullRequest", trigger.Component)
	assert.Equal(t, []any{"created", "updated"}, trigger.Configuration["actions"])

	wait := findSpecNode(t, canvas, prFeedbackWaitChecksNodeID)
	assert.Equal(t, "bitbucket.waitForBuilds", wait.Component)
	assert.Equal(t, []any{"build-a"}, wait.Configuration["buildKeys"])
	_, hasCheckNames := wait.Configuration["checkNames"]
	assert.False(t, hasCheckNames)

	for _, node := range canvas.Spec.Nodes {
		assert.NotContains(t, strings.ToLower(node.Component), "github")
	}
	encoded, err := goyaml.Marshal(canvas)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "GITHUB_TOKEN")
	assert.NotContains(t, string(encoded), "github.com")
	assert.Contains(t, string(encoded), "BITBUCKET_TOKEN")
	assert.Contains(t, string(encoded), "api.bitbucket.org")
}

func TestPrFeedbackChecksPassedDescriptionExpression(t *testing.T) {
	got := evalWaitChecksExpression(t, prFeedbackChecksPassedDescriptionExpression(), map[string]any{
		"selectedChecks": []any{
			map[string]any{
				"name":        "ci/semaphoreci/push",
				"description": "CI",
				"detailsUrl":  "https://example.com/ci",
			},
			map[string]any{
				"name":       "GitGuardian",
				"detailsUrl": "https://example.com/gg",
			},
		},
	})
	assert.Equal(
		t,
		"· [ci/semaphoreci/push: CI](https://example.com/ci)\n"+
			"· [GitGuardian](https://example.com/gg)",
		got,
	)
}

func TestPrFeedbackChecksPassedDescriptionExpression_OmitsEmptyParts(t *testing.T) {
	got := evalWaitChecksExpression(t, prFeedbackChecksPassedDescriptionExpression(), map[string]any{
		"selectedChecks": []any{
			map[string]any{"name": "lint"},
		},
	})
	assert.Equal(t, "· lint", got)
}

func TestPrFeedbackChecksRepairDescriptionExpression(t *testing.T) {
	got := evalWaitChecksExpression(t, prFeedbackChecksRepairDescriptionExpression(), map[string]any{
		"failedChecks": []any{
			map[string]any{
				"name":        "ci/semaphoreci/push",
				"description": "CI",
				"detailsUrl":  "https://example.com/ci",
				"summary":     "The build failed on Semaphore 2.0.",
			},
		},
	})
	assert.Equal(
		t,
		"Failed checks\n"+
			"· [ci/semaphoreci/push: CI](https://example.com/ci): The build failed on Semaphore 2.0.",
		got,
	)
}

func checksTitleData() map[string]any {
	return map[string]any{
		"repository": map[string]any{
			"full_name": "acme/app",
			"html_url":  "https://github.com/acme/app",
		},
		"pull_request": map[string]any{
			"head": map[string]any{
				"sha": "3fc0c4c0123456789abcdef",
			},
		},
	}
}

func evalWaitChecksExpression(t *testing.T, wrapped string, waitData map[string]any) string {
	t.Helper()

	source := templateExpressionSource(t, wrapped)
	env := map[string]any{
		"$": map[string]any{
			prFeedbackWaitChecksNodeName: map[string]any{
				"data": waitData,
			},
		},
	}
	program, err := expr.Compile(source, expr.AsAny(), expr.Env(env), expr.AllowUndefinedVariables())
	require.NoError(t, err)
	got, err := expr.Run(program, env)
	require.NoError(t, err)
	text, ok := got.(string)
	require.True(t, ok)
	return text
}
