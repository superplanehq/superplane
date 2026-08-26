package factories

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

type prFeedbackGraph struct {
	CommentTriggerNodeID string
	ReviewTriggerNodeID  string
	ReplyTriggerNodeID   string
	FindNodeID           string
	RunnerNodeID         string
}

func (g prFeedbackGraph) triggerNodeIDs() []string {
	return []string{g.CommentTriggerNodeID, g.ReviewTriggerNodeID, g.ReplyTriggerNodeID}
}

func (g prFeedbackGraph) Healthy(spec models.LiveCanvasSpec) bool {
	if g.CommentTriggerNodeID == "" || g.ReviewTriggerNodeID == "" || g.ReplyTriggerNodeID == "" {
		return false
	}
	if g.FindNodeID == "" || g.RunnerNodeID == "" {
		return false
	}

	for _, triggerID := range g.triggerNodeIDs() {
		if !hasCanvasPath(spec.Edges, triggerID, g.FindNodeID) {
			return false
		}
	}
	if !hasCanvasPath(spec.Edges, g.FindNodeID, g.RunnerNodeID) {
		return false
	}

	runner := findIntakeNode(spec.Nodes, g.RunnerNodeID)
	if runner == nil || !slices.Contains(intakeAnalysisComponents, runner.ComponentName()) {
		return false
	}

	for _, triggerID := range g.triggerNodeIDs() {
		if strings.TrimSpace(prFeedbackNodeString(findIntakeNode(spec.Nodes, triggerID), "repository")) == "" {
			return false
		}
	}

	return true
}

func resolvePRFeedbackGraph(spec models.LiveCanvasSpec) prFeedbackGraph {
	nodes := spec.Nodes
	return prFeedbackGraph{
		CommentTriggerNodeID: resolveIntakeNode(nodes, prFeedbackCommentTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRComment"
		}),
		ReviewTriggerNodeID: resolveIntakeNode(nodes, prFeedbackReviewTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRReview"
		}),
		ReplyTriggerNodeID: resolveIntakeNode(nodes, prFeedbackReplyTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRReviewComment"
		}),
		FindNodeID: resolveIntakeNode(nodes, prFeedbackFindNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackFindComponent
		}),
		RunnerNodeID: resolveIntakeNode(nodes, prFeedbackRunnerNodeID, func(node *models.Node) bool {
			return slices.Contains(intakeAnalysisComponents, node.ComponentName())
		}),
	}
}

func prFeedbackNodeString(node *models.Node, key string) string {
	if node == nil {
		return ""
	}
	value, _ := node.Configuration[key].(string)
	return value
}

func prFeedbackNodeBool(node *models.Node, key string, fallback bool) bool {
	if node == nil {
		return fallback
	}
	value, ok := node.Configuration[key].(bool)
	if !ok {
		return fallback
	}
	return value
}
