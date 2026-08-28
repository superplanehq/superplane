package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func Test__ResolvePRFeedbackGraph(t *testing.T) {
	t.Run("resolves a generated graph", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "acme/app")

		graph := resolvePRFeedbackGraph(spec)
		assert.Equal(t, prFeedbackCommentTriggerNodeID, graph.CommentTriggerNodeID)
		assert.Equal(t, prFeedbackReviewTriggerNodeID, graph.ReviewTriggerNodeID)
		assert.Equal(t, prFeedbackReplyTriggerNodeID, graph.ReplyTriggerNodeID)
		assert.Equal(t, prFeedbackFindNodeID, graph.FindNodeID)
		assert.Equal(t, prFeedbackActivityNodeID, graph.ActivityNodeID)
		assert.Equal(t, prFeedbackRunnerNodeID, graph.RunnerNodeID)
		assert.True(t, graph.Healthy(spec))
	})

	t.Run("a graph without a runner is not healthy", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(prFeedbackCommentTriggerNodeID, "github.onPRComment"),
				triggerNode(prFeedbackReviewTriggerNodeID, "github.onPRReview"),
				triggerNode(prFeedbackReplyTriggerNodeID, "github.onPRReviewComment"),
				componentNode(prFeedbackFindNodeID, prFeedbackFindComponent),
				componentNode(prFeedbackActivityNodeID, prFeedbackActivityComponent),
			},
			Edges: []models.Edge{
				{SourceID: prFeedbackCommentTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{SourceID: prFeedbackReviewTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{SourceID: prFeedbackReplyTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{SourceID: prFeedbackFindNodeID, TargetID: prFeedbackActivityNodeID},
			},
		}

		graph := resolvePRFeedbackGraph(spec)
		assert.Empty(t, graph.RunnerNodeID)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("a disconnected graph is not healthy", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "acme/app")
		spec.Edges = nil

		graph := resolvePRFeedbackGraph(spec)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("a missing repository binding is not healthy", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "")

		graph := resolvePRFeedbackGraph(spec)
		assert.False(t, graph.Healthy(spec))
	})
}

func Test__BuildPRFeedbackCanvas(t *testing.T) {
	t.Run("the mention starts one run from comment, review, or reply", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    prFeedbackDefaultMention,
			IgnoreBots: true,
		})

		assert.Equal(t, []string{
			"default:" + prFeedbackCommentTriggerNodeID + "->" + prFeedbackFindNodeID,
			"default:" + prFeedbackReviewTriggerNodeID + "->" + prFeedbackFindNodeID,
			"default:" + prFeedbackReplyTriggerNodeID + "->" + prFeedbackFindNodeID,
			"found:" + prFeedbackFindNodeID + "->" + prFeedbackActivityNodeID,
			"default:" + prFeedbackActivityNodeID + "->" + prFeedbackRunnerNodeID,
		}, yamlEdgeChannels(canvas))

		for _, node := range canvas.Spec.Nodes {
			assert.NotEqual(t, "noop", node.Component)
			assert.NotEqual(t, "finish", node.ID)
		}

		reply := findSpecNode(t, canvas, prFeedbackReplyTriggerNodeID)
		assert.Equal(t, false, reply.Configuration["includeReviewSubmissions"])
		assert.Equal(t, prFeedbackCommentScopeReplies, reply.Configuration["commentScope"])
		assert.Equal(t, true, reply.Configuration["ignoreBots"])
		assert.Equal(t, prFeedbackDefaultMention, reply.Configuration["contentFilter"])

		activity := findSpecNode(t, canvas, prFeedbackActivityNodeID)
		assert.Equal(t, `{{ $["Find Pull Request"].data.pullRequest.id }}`, activity.Configuration["pullRequestId"])
		assert.Equal(t, prFeedbackActivityDescriptionExpression(), activity.Configuration["description"])
	})

	t.Run("the allowed bots list is applied to every trigger", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:  "acme/app",
			Mention:     prFeedbackDefaultMention,
			IgnoreBots:  true,
			AllowedBots: []string{"coderabbitai", "bugbot"},
		})

		for _, nodeID := range []string{prFeedbackCommentTriggerNodeID, prFeedbackReviewTriggerNodeID, prFeedbackReplyTriggerNodeID} {
			node := findSpecNode(t, canvas, nodeID)
			assert.Equal(t, []any{"coderabbitai", "bugbot"}, node.Configuration["allowedBots"])
		}
	})

	t.Run("an empty allowed bots list is omitted from trigger configuration", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    prFeedbackDefaultMention,
			IgnoreBots: true,
		})

		for _, nodeID := range []string{prFeedbackCommentTriggerNodeID, prFeedbackReviewTriggerNodeID, prFeedbackReplyTriggerNodeID} {
			node := findSpecNode(t, canvas, nodeID)
			_, exists := node.Configuration["allowedBots"]
			assert.False(t, exists)
		}
	})

	t.Run("the runner checks out the pull request head branch", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    prFeedbackDefaultMention,
			IgnoreBots: true,
		})
		runner := findSpecNode(t, canvas, prFeedbackRunnerNodeID)
		assert.Contains(t, runnerEnv(t, runner, "PR_HEAD"), "pull_request?.head?.ref")

		checkout := runnerStepCommand(t, runner, "Checkout Pull Request")
		assert.Contains(t, checkout, `git fetch origin "pull/${PR_NUMBER}/head:${PR_HEAD}"`)
		assert.Contains(t, checkout, `git checkout "${PR_HEAD}"`)
		assert.NotContains(t, checkout, "pr-feedback")

		assert.Contains(t, runnerEnv(t, runner, "COAUTHORS"), "order().assignees")
		dco := runnerStepCommand(t, runner, "Set Up DCO Signing")
		assert.Contains(t, dco, `${COAUTHORS:-}`)
	})

	t.Run("the runner names the model it runs", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Agent: &intakeAgent{
				Component:   "runnerClaudeCode",
				Credentials: map[string]any{"source": "integration"},
			},
		})

		runner := findSpecNode(t, canvas, prFeedbackRunnerNodeID)
		assert.Equal(t, "opus", runner.Configuration["model"])
	})
}

func yamlEdgeChannels(canvas *yaml.Canvas) []string {
	result := make([]string, 0, len(canvas.Spec.Edges))
	for _, edge := range canvas.Spec.Edges {
		result = append(result, edge.Channel+":"+edge.SourceID+"->"+edge.TargetID)
	}
	return result
}

func runnerEnv(t *testing.T, node yaml.Node, name string) string {
	t.Helper()

	entries, ok := node.Configuration["environment"].([]any)
	require.True(t, ok, "runner has no environment")
	for _, entry := range entries {
		item, ok := entry.(map[string]any)
		require.True(t, ok)
		if item["name"] == name {
			value, _ := item["value"].(string)
			return value
		}
	}
	require.Failf(t, "environment variable not found", "runner has no %q", name)
	return ""
}

func runnerStepCommand(t *testing.T, node yaml.Node, name string) string {
	t.Helper()

	steps, ok := node.Configuration["steps"].([]any)
	require.True(t, ok, "runner has no steps")
	for _, step := range steps {
		item, ok := step.(map[string]any)
		require.True(t, ok)
		if item["name"] == name {
			command, ok := item["command"].(string)
			require.True(t, ok, "step %q has no command", name)
			return command
		}
	}
	require.Failf(t, "step not found", "runner has no step %q", name)
	return ""
}

func prFeedbackSpecFromTemplate(t *testing.T, repository string) models.LiveCanvasSpec {
	t.Helper()

	canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository: repository,
		Mention:    prFeedbackDefaultMention,
		IgnoreBots: true,
	})
	return models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
}
