package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
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
		assert.Equal(t, []resolvedPRFeedbackDiscussionFlow{
			{
				TriggerNodeID:  prFeedbackCommentTriggerNodeID,
				FindNodeID:     prFeedbackFindNodeID,
				ActivityNodeID: prFeedbackActivityNodeID,
				RunnerNodeID:   prFeedbackRunnerNodeID,
			},
			{
				TriggerNodeID:  prFeedbackReviewTriggerNodeID,
				FindNodeID:     prFeedbackReviewFindNodeID,
				ActivityNodeID: prFeedbackReviewActivityNodeID,
				RunnerNodeID:   prFeedbackReviewRunnerNodeID,
			},
			{
				TriggerNodeID:  prFeedbackReplyTriggerNodeID,
				FindNodeID:     prFeedbackReplyFindNodeID,
				ActivityNodeID: prFeedbackReplyActivityNodeID,
				RunnerNodeID:   prFeedbackReplyRunnerNodeID,
			},
		}, graph.discussionFlows(spec))
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

	t.Run("a canvas without a comment reaction is still healthy", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "acme/app")
		spec = withoutPRFeedbackAcknowledgeComment(spec)

		graph := resolvePRFeedbackGraph(spec)
		assert.True(t, graph.Healthy(spec))
		assert.Equal(t, []resolvedPRFeedbackDiscussionFlow{
			{
				TriggerNodeID:  prFeedbackCommentTriggerNodeID,
				FindNodeID:     prFeedbackFindNodeID,
				ActivityNodeID: prFeedbackActivityNodeID,
				RunnerNodeID:   prFeedbackRunnerNodeID,
			},
			{
				TriggerNodeID:  prFeedbackReviewTriggerNodeID,
				FindNodeID:     prFeedbackReviewFindNodeID,
				ActivityNodeID: prFeedbackReviewActivityNodeID,
				RunnerNodeID:   prFeedbackReviewRunnerNodeID,
			},
			{
				TriggerNodeID:  prFeedbackReplyTriggerNodeID,
				FindNodeID:     prFeedbackReplyFindNodeID,
				ActivityNodeID: prFeedbackReplyActivityNodeID,
				RunnerNodeID:   prFeedbackReplyRunnerNodeID,
			},
		}, graph.discussionFlows(spec))
	})

	t.Run("resolves a generated checks graph", func(t *testing.T) {
		spec := prFeedbackChecksSpecFromTemplate(t, "acme/app")

		graph := resolvePRFeedbackGraph(spec)
		assert.Equal(t, prFeedbackPullRequestTriggerNodeID, graph.PullRequestTriggerNodeID)
		assert.Equal(t, prFeedbackWaitChecksNodeID, graph.WaitChecksNodeID)
		assert.Equal(t, prFeedbackStartRepairNodeID, graph.StartRepairNodeID)
		assert.True(t, graph.isChecks())
		assert.True(t, graph.Healthy(spec))
		for _, node := range spec.Nodes {
			assert.NotEqual(t, "setWorkOrderStatusNote", node.ComponentName())
		}
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
			"found:" + prFeedbackFindNodeID + "->" + prFeedbackActivityNodeID,
			"default:" + prFeedbackActivityNodeID + "->" + prFeedbackRunnerNodeID,
			"default:" + prFeedbackCommentTriggerNodeID + "->" + prFeedbackAcknowledgeCommentNodeID,
			"default:" + prFeedbackReviewTriggerNodeID + "->" + prFeedbackReviewFindNodeID,
			"found:" + prFeedbackReviewFindNodeID + "->" + prFeedbackReviewActivityNodeID,
			"default:" + prFeedbackReviewActivityNodeID + "->" + prFeedbackReviewRunnerNodeID,
			"default:" + prFeedbackReviewTriggerNodeID + "->" + prFeedbackAcknowledgeReviewNodeID,
			"default:" + prFeedbackReplyTriggerNodeID + "->" + prFeedbackReplyFindNodeID,
			"found:" + prFeedbackReplyFindNodeID + "->" + prFeedbackReplyActivityNodeID,
			"default:" + prFeedbackReplyActivityNodeID + "->" + prFeedbackReplyRunnerNodeID,
			"default:" + prFeedbackReplyTriggerNodeID + "->" + prFeedbackAcknowledgeReviewReplyNodeID,
		}, yamlEdgeChannels(canvas))

		for _, node := range canvas.Spec.Nodes {
			assert.NotEqual(t, "noop", node.Component)
			assert.NotEqual(t, "finish", node.ID)
		}

		assertPRFeedbackAcknowledgeNode(t, canvas, prFeedbackAcknowledgeCommentNodeID,
			prFeedbackCommentAcknowledgeCommentIDExpression(), "issueComment",
			prFeedbackAcknowledgePosition(prFeedbackCommentFlowY))
		assertPRFeedbackAcknowledgeNode(t, canvas, prFeedbackAcknowledgeReviewNodeID,
			prFeedbackReviewAcknowledgeCommentIDExpression(), "reviewComment",
			prFeedbackAcknowledgePosition(prFeedbackReviewFlowY))
		assertPRFeedbackAcknowledgeNode(t, canvas, prFeedbackAcknowledgeReviewReplyNodeID,
			prFeedbackCommentAcknowledgeCommentIDExpression(), "reviewComment",
			prFeedbackAcknowledgePosition(prFeedbackReplyFlowY))
		var reactionIDs []string
		for _, node := range canvas.Spec.Nodes {
			if node.Component == "github.addReaction" {
				reactionIDs = append(reactionIDs, node.ID)
			}
		}
		assert.Equal(t, []string{
			prFeedbackAcknowledgeCommentNodeID,
			prFeedbackAcknowledgeReviewNodeID,
			prFeedbackAcknowledgeReviewReplyNodeID,
		}, reactionIDs)

		reply := findSpecNode(t, canvas, prFeedbackReplyTriggerNodeID)
		assert.Equal(t, false, reply.Configuration["includeReviewSubmissions"])
		assert.Equal(t, prFeedbackCommentScopeReplies, reply.Configuration["commentScope"])
		assert.Equal(t, true, reply.Configuration["ignoreBots"])
		assert.Equal(t, prFeedbackDefaultMention, reply.Configuration["contentFilter"])

		activity := findSpecNode(t, canvas, prFeedbackActivityNodeID)
		assert.Equal(t, `{{ $["Find Pull Request"].data.pullRequest.id }}`, activity.Configuration["pullRequestId"])
		assert.Equal(t, prFeedbackCommentActivityTitleExpression(), activity.Configuration["title"])
		assert.Equal(t, prFeedbackCommentActivityDescriptionExpression(), activity.Configuration["description"])

		reviewActivity := findSpecNode(t, canvas, prFeedbackReviewActivityNodeID)
		assert.Equal(t, `{{ $["Find Pull Request For Review"].data.pullRequest.id }}`, reviewActivity.Configuration["pullRequestId"])
		assert.Equal(t, prFeedbackReviewActivityTitleExpression(), reviewActivity.Configuration["title"])
		assert.Equal(t, prFeedbackReviewActivityDescriptionExpression(), reviewActivity.Configuration["description"])

		replyActivity := findSpecNode(t, canvas, prFeedbackReplyActivityNodeID)
		assert.Equal(t, `{{ $["Find Pull Request For Review Reply"].data.pullRequest.id }}`, replyActivity.Configuration["pullRequestId"])
		assert.Equal(t, prFeedbackReplyActivityTitleExpression(), replyActivity.Configuration["title"])
		assert.Equal(t, prFeedbackCommentActivityDescriptionExpression(), replyActivity.Configuration["description"])
	})

	t.Run("discussion runners support visual evidence without a second comment", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    prFeedbackDefaultMention,
			IgnoreBots: true,
		})

		for _, nodeID := range []string{prFeedbackRunnerNodeID, prFeedbackReviewRunnerNodeID, prFeedbackReplyRunnerNodeID} {
			runner := findSpecNode(t, canvas, nodeID)
			assert.Equal(t, false, runner.Configuration["includeVisualEvidence"])
			assert.False(t, runnerHasStep(t, runner, "Publish Visual Evidence"))
		}

		for _, node := range canvas.Spec.Nodes {
			assert.NotEqual(t, "github.createIssueComment", node.Component)
			assert.NotContains(t, node.ID, "visual-evidence")
		}

		prompt := prFeedbackPrompt()
		assert.Contains(t, prompt, "Explain disagreements in the completion comment.")
		assert.Contains(t, prompt, "Post one pull request comment after you address all feedback.")
		assert.Contains(t, prompt, "Summarize the changes in this comment.")
		assert.Contains(t, prompt, "If you upload visual evidence, include it in the same comment.")
		assert.Contains(t, prompt, "Do not post a separate visual evidence comment.")
	})

	t.Run("an empty mention is written as an empty content filter", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    "",
			IgnoreBots: true,
		})

		for _, nodeID := range []string{prFeedbackCommentTriggerNodeID, prFeedbackReviewTriggerNodeID, prFeedbackReplyTriggerNodeID} {
			node := findSpecNode(t, canvas, nodeID)
			assert.Equal(t, "", node.Configuration["contentFilter"])
		}
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
		assert.Contains(t, checkout, "gh auth setup-git --hostname github.com --force")
		assert.Contains(t, checkout, `git clone --depth 1 "https://github.com/${REPO}.git" repo`)
		assert.NotContains(t, checkout, "x-access-token")
		assert.Contains(t, checkout, `git fetch origin "pull/${PR_NUMBER}/head:${PR_HEAD}"`)
		assert.Contains(t, checkout, `git checkout "${PR_HEAD}"`)
		assert.NotContains(t, checkout, "pr-feedback")

		assert.Contains(t, runnerEnv(t, runner, "COAUTHORS"), "task().assignees")
		dco := runnerStepCommand(t, runner, "Set Up DCO Signing")
		assert.Contains(t, dco, `${COAUTHORS:-}`)

		// "git commit -s" signs off and the agent amends commits, so appending
		// the trailers wrote them twice. It also put a blank line before the
		// sign-off, which left it outside the trailer block GitHub reads.
		assert.Contains(t, dco, "--if-exists doNothing")
		assert.Contains(t, dco, "--if-exists addIfDifferent")
		assert.NotContains(t, dco, `>> "$1"`)
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

	t.Run("action nodes serialize per pull request", func(t *testing.T) {
		canvas := buildPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			Mention:    prFeedbackDefaultMention,
			IgnoreBots: true,
		})

		assertPRFeedbackPerPullRequestConcurrency(t, canvas)
	})
}

func Test__EnsurePRFeedbackConcurrency(t *testing.T) {
	t.Run("stamps a per-pull-request key on action nodes that still serialize globally", func(t *testing.T) {
		nodes := []models.Node{
			triggerNode(prFeedbackCommentTriggerNodeID, "github.onPRComment"),
			componentNode(prFeedbackRunnerNodeID, "runnerClaudeCode"),
			componentNode(prFeedbackWaitChecksNodeID, prFeedbackWaitChecksComponent),
		}

		nodes = ensurePRFeedbackConcurrency(nodes)

		assert.Nil(t, nodes[0].Concurrency)
		require.NotNil(t, nodes[1].Concurrency)
		assert.Equal(t, prFeedbackConcurrencyKey, nodes[1].Concurrency.Key)
		assert.Nil(t, nodes[1].Concurrency.Max)
		require.NotNil(t, nodes[2].Concurrency)
		assert.Equal(t, prFeedbackConcurrencyKey, nodes[2].Concurrency.Key)
	})

	t.Run("keeps a concurrency spec that is already set", func(t *testing.T) {
		max := 4
		nodes := []models.Node{
			componentNode(prFeedbackRunnerNodeID, "runnerClaudeCode"),
		}
		nodes[0].Concurrency = &models.ConcurrencySpec{Max: &max, Key: "custom"}

		nodes = ensurePRFeedbackConcurrency(nodes)

		require.NotNil(t, nodes[0].Concurrency)
		assert.Equal(t, "custom", nodes[0].Concurrency.Key)
		require.NotNil(t, nodes[0].Concurrency.Max)
		assert.Equal(t, 4, *nodes[0].Concurrency.Max)
	})
}

func Test__BuildChecksPRFeedbackCanvas(t *testing.T) {
	t.Run("opened, reopened, and synchronize start one wait then one repair", func(t *testing.T) {
		canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			MaximumAttempts: prFeedbackDefaultMaximumAttempts,
			CheckNames:      []string{"lint", "unit"},
		})

		assert.Equal(t, []string{
			"default:" + prFeedbackPullRequestTriggerNodeID + "->" + prFeedbackFindNodeID,
			"found:" + prFeedbackFindNodeID + "->" + prFeedbackActivityNodeID,
			"default:" + prFeedbackActivityNodeID + "->" + prFeedbackWaitChecksNodeID,
			"passed:" + prFeedbackWaitChecksNodeID + "->" + prFeedbackMarkPassedNodeID,
			"failed:" + prFeedbackWaitChecksNodeID + "->" + prFeedbackStartRepairNodeID,
			"timedOut:" + prFeedbackWaitChecksNodeID + "->" + prFeedbackStopWaitingNodeID,
			"default:" + prFeedbackStartRepairNodeID + "->" + prFeedbackRunnerNodeID,
			"limitReached:" + prFeedbackStartRepairNodeID + "->" + prFeedbackPauseFixesNodeID,
			"default:" + prFeedbackStopWaitingNodeID + "->" + prFeedbackRecordTimeoutNodeID,
		}, yamlEdgeChannels(canvas))

		activity := findSpecNode(t, canvas, prFeedbackActivityNodeID)
		assert.Equal(t, "concurrent", activity.Configuration["access"])
		assert.Equal(t, prFeedbackPRHeadSHAExpression(), activity.Configuration["revision"])
		assert.Equal(t, prFeedbackChecksWaitingTitleExpression(), activity.Configuration["title"])
		assert.Nil(t, activity.Configuration["description"])

		passed := findSpecNode(t, canvas, prFeedbackMarkPassedNodeID)
		assert.Equal(t, prFeedbackChecksPassedTitleExpression(), passed.Configuration["title"])
		assert.Equal(t, prFeedbackChecksPassedDescriptionExpression(), passed.Configuration["description"])

		repair := findSpecNode(t, canvas, prFeedbackStartRepairNodeID)
		assert.Equal(t, prFeedbackChecksRepairTitleExpression(), repair.Configuration["title"])
		assert.Equal(t, prFeedbackChecksRepairDescriptionExpression(), repair.Configuration["description"])

		wait := findSpecNode(t, canvas, prFeedbackWaitChecksNodeID)
		assert.Equal(t, []any{"lint", "unit"}, wait.Configuration["checkNames"])

		pause := findSpecNode(t, canvas, prFeedbackPauseFixesNodeID)
		assert.Equal(t, "Automatic fixes paused after 3 attempts", pause.Configuration["title"])

		for _, node := range canvas.Spec.Nodes {
			assert.NotEqual(t, "setWorkOrderStatusNote", node.Component)
		}
	})

	t.Run("the runner verifies the remote head before it pushes", func(t *testing.T) {
		canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			MaximumAttempts: prFeedbackDefaultMaximumAttempts,
		})
		runner := findSpecNode(t, canvas, prFeedbackRunnerNodeID)
		assert.Contains(t, runnerEnv(t, runner, "FAILED_CHECKS"), "Wait For Pull Request Checks")
		assert.Contains(t, runnerEnv(t, runner, "PR_REVISION"), "pull_request.head.sha")

		push := runnerStepCommand(t, runner, "Commit and Push")
		assert.Contains(t, push, "REMOTE_HEAD")
		assert.Contains(t, push, `if [ "${REMOTE_HEAD}" != "${PR_REVISION}" ]`)
		assert.Contains(t, push, `git commit -s -m "fix: repair failing checks on PR #${PR_NUMBER}"`)

		checkout := runnerStepCommand(t, runner, "Checkout Pull Request")
		assert.Contains(t, checkout, "gh auth setup-git --hostname github.com --force")
		assert.Contains(t, checkout, `git clone --depth 1 "https://github.com/${REPO}.git" repo`)
		assert.NotContains(t, checkout, "x-access-token")
		assert.Equal(t, false, runner.Configuration["includeVisualEvidence"])
	})

	t.Run("action nodes serialize per pull request", func(t *testing.T) {
		canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			MaximumAttempts: prFeedbackDefaultMaximumAttempts,
		})

		assertPRFeedbackPerPullRequestConcurrency(t, canvas)
	})
}

func assertPRFeedbackPerPullRequestConcurrency(t *testing.T, canvas *yaml.Canvas) {
	t.Helper()

	for _, node := range canvas.Spec.Nodes {
		if node.Type == yaml.NodeTypeTrigger {
			assert.Nilf(t, node.Concurrency, "trigger %s caps its concurrency", node.ID)
			continue
		}

		require.NotNilf(t, node.Concurrency, "node %s has no concurrency", node.ID)
		assert.Equalf(t, prFeedbackConcurrencyKey, node.Concurrency.Key, "node %s", node.ID)
		assert.Nilf(t, node.Concurrency.Max, "node %s sets a concurrency max", node.ID)
	}
}

func assertPRFeedbackAcknowledgeNode(
	t *testing.T,
	canvas *yaml.Canvas,
	nodeID string,
	commentID string,
	target string,
	position yaml.Position,
) {
	t.Helper()

	node := findSpecNode(t, canvas, nodeID)
	assert.Equal(t, "github.addReaction", node.Component)
	assert.Equal(t, "{{ root().data.repository.full_name }}", node.Configuration["repository"])
	assert.Equal(t, commentID, node.Configuration["commentId"])
	assert.Equal(t, "eyes", node.Configuration["content"])
	assert.Equal(t, target, node.Configuration["target"])
	assert.Equal(t, position, node.Position)
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

func runnerHasStep(t *testing.T, node yaml.Node, name string) bool {
	t.Helper()

	steps, ok := node.Configuration["steps"].([]any)
	require.True(t, ok, "runner has no steps")
	for _, step := range steps {
		item, ok := step.(map[string]any)
		require.True(t, ok)
		if item["name"] == name {
			return true
		}
	}
	return false
}

func withoutPRFeedbackAcknowledgeComment(spec models.LiveCanvasSpec) models.LiveCanvasSpec {
	nodes := make([]models.Node, 0, len(spec.Nodes))
	for _, node := range spec.Nodes {
		if node.ID == prFeedbackAcknowledgeCommentNodeID {
			continue
		}
		nodes = append(nodes, node)
	}
	edges := make([]models.Edge, 0, len(spec.Edges))
	for _, edge := range spec.Edges {
		if edge.TargetID == prFeedbackAcknowledgeCommentNodeID {
			continue
		}
		edges = append(edges, edge)
	}
	spec.Nodes = nodes
	spec.Edges = edges
	return spec
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

func prFeedbackChecksSpecFromTemplate(t *testing.T, repository string) models.LiveCanvasSpec {
	t.Helper()

	canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository:      repository,
		MaximumAttempts: prFeedbackDefaultMaximumAttempts,
	})
	return models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
}

func requireValidTemplateExpressions(t *testing.T, value string) {
	t.Helper()

	matches := configuration.ExpressionPlaceholderRegex.FindAllString(value, -1)
	require.NotEmpty(t, matches)
	for _, match := range matches {
		source := match[2 : len(match)-2]
		require.NoError(t, expressionvalidation.ValidateExpression(source, nil), match)
	}
}
