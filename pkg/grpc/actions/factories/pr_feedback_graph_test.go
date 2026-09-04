package factories

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func Test__ResolvePRFeedbackGraph(t *testing.T) {
	t.Run("resolves a generated graph", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "acme/app")

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, spec)
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

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, spec)
		assert.Empty(t, graph.RunnerNodeID)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("a disconnected graph is not healthy", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "acme/app")
		spec.Edges = nil

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, spec)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("a missing repository binding is not healthy", func(t *testing.T) {
		spec := prFeedbackSpecFromTemplate(t, "")

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion, spec)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("resolves a generated checks graph", func(t *testing.T) {
		spec := prFeedbackChecksSpecFromTemplate(t, "acme/app")

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestChecks, spec)
		assert.Equal(t, prFeedbackPullRequestTriggerNodeID, graph.PullRequestTriggerNodeID)
		assert.Equal(t, prFeedbackWaitChecksNodeID, graph.WaitChecksNodeID)
		assert.Equal(t, prFeedbackStartRepairNodeID, graph.StartRepairNodeID)
		assert.Equal(t, prFeedbackAnnounceLimitNodeID, graph.AnnounceLimitNodeID)
		assert.True(t, graph.isChecks())
		assert.True(t, graph.Healthy(spec))
	})

	t.Run("resolves a generated conflicts graph", func(t *testing.T) {
		spec := prFeedbackConflictsSpecFromTemplate(t, "acme/app", "main")

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestConflicts, spec)
		assert.Equal(t, prFeedbackPullRequestTriggerNodeID, graph.PullRequestTriggerNodeID)
		assert.Equal(t, prFeedbackPushTriggerNodeID, graph.PushTriggerNodeID)
		assert.Equal(t, prFeedbackFindNodeID, graph.FindNodeID)
		assert.Equal(t, prFeedbackListNodeID, graph.ListNodeID)
		assert.Equal(t, prFeedbackForEachNodeID, graph.ForEachNodeID)
		assert.Equal(t, prFeedbackWaitMergeableNodeID, graph.WaitMergeableNodeID)
		assert.Equal(t, prFeedbackStartConflictRepairNodeID, graph.ActivityNodeID)
		assert.Equal(t, prFeedbackConflictsRunnerNodeID, graph.RunnerNodeID)
		assert.True(t, graph.isConflicts())
		assert.True(t, graph.Healthy(spec))
	})

	t.Run("a conflicts graph without the wait node is not healthy", func(t *testing.T) {
		spec := prFeedbackConflictsSpecFromTemplate(t, "acme/app", "main")
		spec.Nodes = slices.DeleteFunc(spec.Nodes, func(node models.Node) bool {
			return node.ID == prFeedbackWaitMergeableNodeID
		})

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestConflicts, spec)
		assert.Empty(t, graph.WaitMergeableNodeID)
		assert.False(t, graph.Healthy(spec))
	})

	t.Run("a conflicts graph without the list path is not healthy", func(t *testing.T) {
		spec := prFeedbackConflictsSpecFromTemplate(t, "acme/app", "main")
		spec.Nodes = slices.DeleteFunc(spec.Nodes, func(node models.Node) bool {
			return node.ID == prFeedbackListNodeID
		})

		graph := resolvePRFeedbackGraph(models.FactoryPRFeedbackHandlerSourcePullRequestConflicts, spec)
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
}

func Test__EnsureChecksAnnounceLimitNode(t *testing.T) {
	t.Run("adds the status note node when a checks graph is missing it", func(t *testing.T) {
		nodes := []models.Node{
			componentNode(prFeedbackPauseFixesNodeID, prFeedbackUpdateActivityComponent),
		}
		edges := []models.Edge{}
		graph := prFeedbackGraph{
			Source:                   models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
			PullRequestTriggerNodeID: prFeedbackPullRequestTriggerNodeID,
			PauseFixesNodeID:         prFeedbackPauseFixesNodeID,
		}

		nodes, edges = ensureChecksAnnounceLimitNode(nodes, edges, graph, 2)

		require.Len(t, nodes, 2)
		assert.Equal(t, prFeedbackAnnounceLimitNodeID, nodes[1].ID)
		assert.Equal(t, prFeedbackSetStatusNoteComponent, nodes[1].ComponentName())
		assert.Equal(t, prFeedbackChecksLimitStatusNoteBody(2), nodes[1].Configuration["body"])
		require.Len(t, edges, 1)
		assert.Equal(t, prFeedbackPauseFixesNodeID, edges[0].SourceID)
		assert.Equal(t, prFeedbackAnnounceLimitNodeID, edges[0].TargetID)
	})

	t.Run("does not duplicate the status note node", func(t *testing.T) {
		nodes := []models.Node{
			componentNode(prFeedbackPauseFixesNodeID, prFeedbackUpdateActivityComponent),
			componentNode(prFeedbackAnnounceLimitNodeID, prFeedbackSetStatusNoteComponent),
		}
		edges := []models.Edge{{
			Channel:  "default",
			SourceID: prFeedbackPauseFixesNodeID,
			TargetID: prFeedbackAnnounceLimitNodeID,
		}}
		graph := prFeedbackGraph{
			Source:                   models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
			PullRequestTriggerNodeID: prFeedbackPullRequestTriggerNodeID,
			PauseFixesNodeID:         prFeedbackPauseFixesNodeID,
			AnnounceLimitNodeID:      prFeedbackAnnounceLimitNodeID,
		}

		nextNodes, nextEdges := ensureChecksAnnounceLimitNode(nodes, edges, graph, 3)

		assert.Len(t, nextNodes, 2)
		assert.Len(t, nextEdges, 1)
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
			"default:" + prFeedbackPauseFixesNodeID + "->" + prFeedbackAnnounceLimitNodeID,
			"default:" + prFeedbackStopWaitingNodeID + "->" + prFeedbackRecordTimeoutNodeID,
		}, yamlEdgeChannels(canvas))

		activity := findSpecNode(t, canvas, prFeedbackActivityNodeID)
		assert.Equal(t, "concurrent", activity.Configuration["access"])
		assert.Equal(t, prFeedbackPRHeadSHAExpression(), activity.Configuration["revision"])

		wait := findSpecNode(t, canvas, prFeedbackWaitChecksNodeID)
		assert.Equal(t, []any{"lint", "unit"}, wait.Configuration["checkNames"])

		pause := findSpecNode(t, canvas, prFeedbackPauseFixesNodeID)
		assert.Equal(t, "Automatic fixes paused after 3 attempts", pause.Configuration["description"])

		note := findSpecNode(t, canvas, prFeedbackAnnounceLimitNodeID)
		assert.Equal(t, prFeedbackSetStatusNoteComponent, note.Component)
		assert.Equal(t, prFeedbackStatusNoteKey, note.Configuration["noteKey"])
		assert.Equal(t, prFeedbackWorkOrderIDExpression(), note.Configuration["orderId"])
		assert.Equal(t, "Automatic fixes did not succeed", note.Configuration["headline"])
		assert.Equal(t, prFeedbackChecksLimitStatusNoteBody(3), note.Configuration["body"])
		assert.Equal(t, true, note.Configuration["showOnlyWhenWaiting"])
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
	})
}

func Test__BuildConflictsPRFeedbackCanvas(t *testing.T) {
	t.Run("the PR-event path and the push path join at the wait node", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			BaseBranch:      "main",
			MaximumAttempts: prFeedbackDefaultMaximumAttempts,
		})

		assert.Equal(t, []string{
			"default:" + prFeedbackPullRequestTriggerNodeID + "->" + prFeedbackFindNodeID,
			"found:" + prFeedbackFindNodeID + "->" + prFeedbackWaitMergeableNodeID,
			"default:" + prFeedbackPushTriggerNodeID + "->" + prFeedbackListNodeID,
			"default:" + prFeedbackListNodeID + "->" + prFeedbackForEachNodeID,
			"item:" + prFeedbackForEachNodeID + "->" + prFeedbackWaitMergeableNodeID,
			"conflicted:" + prFeedbackWaitMergeableNodeID + "->" + prFeedbackStartConflictRepairNodeID,
			"default:" + prFeedbackStartConflictRepairNodeID + "->" + prFeedbackConflictsRunnerNodeID,
			"limitReached:" + prFeedbackStartConflictRepairNodeID + "->" + prFeedbackPauseFixesNodeID,
			"default:" + prFeedbackPauseFixesNodeID + "->" + prFeedbackAnnounceLimitNodeID,
		}, yamlEdgeChannels(canvas))

		// Clean and timed-out waits stay silent: no node consumes those
		// channels anywhere in the canvas. The edges above already show
		// find and forEach reach the wait node directly, with no activity
		// node in between.
		for _, edge := range canvas.Spec.Edges {
			assert.NotEqual(t, "clean", edge.Channel)
			assert.NotEqual(t, "timedOut", edge.Channel)
		}
	})

	t.Run("the push trigger refs target the configured base branch", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			BaseBranch: "develop",
		})

		push := findSpecNode(t, canvas, prFeedbackPushTriggerNodeID)
		assert.Equal(t, []any{
			map[string]any{"type": "equals", "value": "refs/heads/develop"},
		}, push.Configuration["refs"])
	})

	t.Run("an empty base branch defaults to main", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
		})

		push := findSpecNode(t, canvas, prFeedbackPushTriggerNodeID)
		assert.Equal(t, []any{
			map[string]any{"type": "equals", "value": "refs/heads/main"},
		}, push.Configuration["refs"])
	})

	t.Run("the list node feeds forEach, which feeds the wait node", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			BaseBranch: "main",
		})

		forEach := findSpecNode(t, canvas, prFeedbackForEachNodeID)
		assert.Equal(t, prFeedbackForEachComponent, forEach.Component)
		assert.Contains(t, forEach.Configuration["arrayExpression"], "List Pull Requests")
		assert.Contains(t, forEach.Configuration["arrayExpression"], "pullRequests")

		list := findSpecNode(t, canvas, prFeedbackListNodeID)
		assert.Equal(t, prFeedbackListComponent, list.Component)
		assert.Equal(t, "acme/app", list.Configuration["repository"])
	})

	t.Run("the exclusive activity binds the wait node's head SHA", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository: "acme/app",
			BaseBranch: "main",
		})

		activity := findSpecNode(t, canvas, prFeedbackStartConflictRepairNodeID)
		assert.Equal(t, prFeedbackActivityComponent, activity.Component)
		assert.Equal(t, "exclusive", activity.Configuration["access"])
		assert.Equal(t, prFeedbackConflictsHeadSHAExpression(), activity.Configuration["revision"])
		assert.Contains(t, activity.Configuration["description"], "Resolving conflicts on")
	})

	t.Run("the runner verifies the remote head and uses the conflicts commit subject", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			BaseBranch:      "main",
			MaximumAttempts: prFeedbackDefaultMaximumAttempts,
		})
		runner := findSpecNode(t, canvas, prFeedbackConflictsRunnerNodeID)
		assert.Contains(t, runnerEnv(t, runner, "PR_REVISION"), "sha")
		assert.Contains(t, runnerEnv(t, runner, "BASE_BRANCH"), prFeedbackWaitMergeableNodeName)

		push := runnerStepCommand(t, runner, "Commit and Push")
		assert.Contains(t, push, "REMOTE_HEAD")
		assert.Contains(t, push, `if [ "${REMOTE_HEAD}" != "${PR_REVISION}" ]`)
		assert.Contains(t, push, `git commit -s -m "fix: resolve merge conflicts on PR #${PR_NUMBER}"`)
	})

	t.Run("the pause and announce nodes use conflict-specific copy", func(t *testing.T) {
		canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
			Repository:      "acme/app",
			BaseBranch:      "main",
			MaximumAttempts: 5,
		})

		pause := findSpecNode(t, canvas, prFeedbackPauseFixesNodeID)
		assert.Equal(t, "Automatic conflict fixes paused after 5 attempts", pause.Configuration["description"])

		note := findSpecNode(t, canvas, prFeedbackAnnounceLimitNodeID)
		assert.Equal(t, "Automatic conflict fixes did not succeed", note.Configuration["headline"])
		assert.Equal(t, prFeedbackConflictsLimitStatusNoteBody(5), note.Configuration["body"])
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

func prFeedbackChecksSpecFromTemplate(t *testing.T, repository string) models.LiveCanvasSpec {
	t.Helper()

	canvas := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository:      repository,
		MaximumAttempts: prFeedbackDefaultMaximumAttempts,
	})
	return models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
}

func prFeedbackConflictsSpecFromTemplate(t *testing.T, repository, baseBranch string) models.LiveCanvasSpec {
	t.Helper()

	canvas := buildConflictsPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository:      repository,
		BaseBranch:      baseBranch,
		MaximumAttempts: prFeedbackDefaultMaximumAttempts,
	})
	return models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
}
