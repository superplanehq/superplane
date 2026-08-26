package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			},
			Edges: []models.Edge{
				{SourceID: prFeedbackCommentTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{SourceID: prFeedbackReviewTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{SourceID: prFeedbackReplyTriggerNodeID, TargetID: prFeedbackFindNodeID},
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
			"found:" + prFeedbackFindNodeID + "->" + prFeedbackRunnerNodeID,
			"notFound:" + prFeedbackFindNodeID + "->" + prFeedbackFinishNodeID,
		}, yamlEdgeChannels(canvas))

		reply := findSpecNode(t, canvas, prFeedbackReplyTriggerNodeID)
		assert.Equal(t, false, reply.Configuration["includeReviewSubmissions"])
		assert.Equal(t, prFeedbackCommentScopeReplies, reply.Configuration["commentScope"])
		assert.Equal(t, true, reply.Configuration["ignoreBots"])
		assert.Equal(t, prFeedbackDefaultMention, reply.Configuration["contentFilter"])
	})
}

func yamlEdgeChannels(canvas *yaml.Canvas) []string {
	result := make([]string, 0, len(canvas.Spec.Edges))
	for _, edge := range canvas.Spec.Edges {
		result = append(result, edge.Channel+":"+edge.SourceID+"->"+edge.TargetID)
	}
	return result
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
