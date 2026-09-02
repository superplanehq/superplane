package factories

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

type prFeedbackGraph struct {
	CommentTriggerNodeID     string
	ReviewTriggerNodeID      string
	ReplyTriggerNodeID       string
	PullRequestTriggerNodeID string
	FindNodeID               string
	ActivityNodeID           string
	WaitChecksNodeID         string
	MarkPassedNodeID         string
	StartRepairNodeID        string
	PauseFixesNodeID         string
	AnnounceLimitNodeID      string
	StopWaitingNodeID        string
	RecordTimeoutNodeID      string
	RunnerNodeID             string
}

func (g prFeedbackGraph) isChecks() bool {
	return g.PullRequestTriggerNodeID != ""
}

func (g prFeedbackGraph) triggerNodeIDs() []string {
	if g.isChecks() {
		return []string{g.PullRequestTriggerNodeID}
	}
	return []string{g.CommentTriggerNodeID, g.ReviewTriggerNodeID, g.ReplyTriggerNodeID}
}

func (g prFeedbackGraph) Healthy(spec models.LiveCanvasSpec) bool {
	if g.isChecks() {
		return g.healthyChecks(spec)
	}
	return g.healthyDiscussion(spec)
}

func (g prFeedbackGraph) healthyDiscussion(spec models.LiveCanvasSpec) bool {
	if g.CommentTriggerNodeID == "" || g.ReviewTriggerNodeID == "" || g.ReplyTriggerNodeID == "" {
		return false
	}
	if g.FindNodeID == "" || g.ActivityNodeID == "" || g.RunnerNodeID == "" {
		return false
	}

	for _, triggerID := range g.triggerNodeIDs() {
		if !hasCanvasPath(spec.Edges, triggerID, g.FindNodeID) {
			return false
		}
	}
	if !hasCanvasPath(spec.Edges, g.FindNodeID, g.ActivityNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.ActivityNodeID, g.RunnerNodeID) {
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

func (g prFeedbackGraph) healthyChecks(spec models.LiveCanvasSpec) bool {
	if g.PullRequestTriggerNodeID == "" || g.FindNodeID == "" || g.ActivityNodeID == "" {
		return false
	}
	if g.WaitChecksNodeID == "" || g.StartRepairNodeID == "" || g.RunnerNodeID == "" {
		return false
	}
	if g.MarkPassedNodeID == "" || g.PauseFixesNodeID == "" || g.StopWaitingNodeID == "" || g.RecordTimeoutNodeID == "" {
		return false
	}
	if g.AnnounceLimitNodeID == "" {
		return false
	}

	if !hasCanvasPath(spec.Edges, g.PullRequestTriggerNodeID, g.FindNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.FindNodeID, g.ActivityNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.ActivityNodeID, g.WaitChecksNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.WaitChecksNodeID, g.MarkPassedNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.WaitChecksNodeID, g.StartRepairNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.WaitChecksNodeID, g.StopWaitingNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.StartRepairNodeID, g.RunnerNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.StartRepairNodeID, g.PauseFixesNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.PauseFixesNodeID, g.AnnounceLimitNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.StopWaitingNodeID, g.RecordTimeoutNodeID) {
		return false
	}

	wait := findIntakeNode(spec.Nodes, g.WaitChecksNodeID)
	if wait == nil || wait.ComponentName() != prFeedbackWaitChecksComponent {
		return false
	}
	runner := findIntakeNode(spec.Nodes, g.RunnerNodeID)
	if runner == nil || !slices.Contains(intakeAnalysisComponents, runner.ComponentName()) {
		return false
	}
	if strings.TrimSpace(prFeedbackNodeString(findIntakeNode(spec.Nodes, g.PullRequestTriggerNodeID), "repository")) == "" {
		return false
	}

	return true
}

func resolvePRFeedbackGraph(spec models.LiveCanvasSpec) prFeedbackGraph {
	nodes := spec.Nodes
	graph := prFeedbackGraph{
		CommentTriggerNodeID: resolveIntakeNode(nodes, prFeedbackCommentTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRComment"
		}),
		ReviewTriggerNodeID: resolveIntakeNode(nodes, prFeedbackReviewTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRReview"
		}),
		ReplyTriggerNodeID: resolveIntakeNode(nodes, prFeedbackReplyTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPRReviewComment"
		}),
		PullRequestTriggerNodeID: resolveIntakeNode(nodes, prFeedbackPullRequestTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPullRequest"
		}),
		FindNodeID: resolveIntakeNode(nodes, prFeedbackFindNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackFindComponent
		}),
		ActivityNodeID: resolveIntakeNode(nodes, prFeedbackActivityNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackActivityComponent
		}),
		WaitChecksNodeID: resolveIntakeNode(nodes, prFeedbackWaitChecksNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackWaitChecksComponent
		}),
		MarkPassedNodeID: resolveIntakeNode(nodes, prFeedbackMarkPassedNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackUpdateActivityComponent
		}),
		StartRepairNodeID: resolveIntakeNode(nodes, prFeedbackStartRepairNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackUpdateActivityComponent
		}),
		PauseFixesNodeID: resolveIntakeNode(nodes, prFeedbackPauseFixesNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackUpdateActivityComponent
		}),
		AnnounceLimitNodeID: resolveIntakeNode(nodes, prFeedbackAnnounceLimitNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackSetStatusNoteComponent
		}),
		StopWaitingNodeID: resolveIntakeNode(nodes, prFeedbackStopWaitingNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackUpdateActivityComponent
		}),
		RecordTimeoutNodeID: resolveIntakeNode(nodes, prFeedbackRecordTimeoutNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackAddRunErrorComponent
		}),
		RunnerNodeID: resolveIntakeNode(nodes, prFeedbackRunnerNodeID, func(node *models.Node) bool {
			return slices.Contains(intakeAnalysisComponents, node.ComponentName())
		}),
	}
	return graph
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

func prFeedbackNodeStringSlice(node *models.Node, key string) []string {
	if node == nil {
		return nil
	}

	items, ok := node.Configuration[key].([]any)
	if !ok {
		if values, ok := node.Configuration[key].([]string); ok {
			return values
		}
		return nil
	}

	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}
