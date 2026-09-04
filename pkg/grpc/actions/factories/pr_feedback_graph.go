package factories

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

// prFeedbackGraph locates the nodes of a generated PR feedback canvas.
// Every source (discussion, checks, conflicts) shares FindNodeID,
// ActivityNodeID, and RunnerNodeID even though what those slots mean
// differs a little per source (e.g. ActivityNodeID is the exclusive
// "start-conflict-repair" node for conflicts). Fields specific to one
// source are documented below.
type prFeedbackGraph struct {
	// Source is the handler's models.FactoryPRFeedbackHandlerSource*
	// value. It drives which Healthy check runs and which trigger set
	// triggerNodeIDs returns; it is not read from the canvas.
	Source string

	// Discussion
	CommentTriggerNodeID string
	ReviewTriggerNodeID  string
	ReplyTriggerNodeID   string

	// Shared: find (or list, for conflicts) -> activity -> runner.
	FindNodeID     string
	ActivityNodeID string
	RunnerNodeID   string

	// Checks
	PullRequestTriggerNodeID string
	WaitChecksNodeID         string
	MarkPassedNodeID         string
	StartRepairNodeID        string
	PauseFixesNodeID         string
	AnnounceLimitNodeID      string
	StopWaitingNodeID        string
	RecordTimeoutNodeID      string

	// Conflicts. PullRequestTriggerNodeID above doubles as the conflicts
	// PR-event trigger (same node id and component as checks).
	PushTriggerNodeID   string
	ListNodeID          string
	ForEachNodeID       string
	WaitMergeableNodeID string
}

func (g prFeedbackGraph) isChecks() bool {
	return g.Source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks
}

func (g prFeedbackGraph) isConflicts() bool {
	return g.Source == models.FactoryPRFeedbackHandlerSourcePullRequestConflicts
}

func (g prFeedbackGraph) triggerNodeIDs() []string {
	switch {
	case g.isConflicts():
		return []string{g.PullRequestTriggerNodeID, g.PushTriggerNodeID}
	case g.isChecks():
		return []string{g.PullRequestTriggerNodeID}
	default:
		return []string{g.CommentTriggerNodeID, g.ReviewTriggerNodeID, g.ReplyTriggerNodeID}
	}
}

func (g prFeedbackGraph) Healthy(spec models.LiveCanvasSpec) bool {
	switch {
	case g.isConflicts():
		return g.healthyConflicts(spec)
	case g.isChecks():
		return g.healthyChecks(spec)
	default:
		return g.healthyDiscussion(spec)
	}
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

// healthyConflicts intentionally does not require the checks wait-activity
// nodes (mark passed, stop waiting, record timeout) or the attempt-limit
// nodes: a conflict canvas has neither, and pause/announce are optional
// instrumentation, not the runtime-critical path.
func (g prFeedbackGraph) healthyConflicts(spec models.LiveCanvasSpec) bool {
	if g.PullRequestTriggerNodeID == "" || g.PushTriggerNodeID == "" {
		return false
	}
	if g.FindNodeID == "" || g.ListNodeID == "" || g.ForEachNodeID == "" {
		return false
	}
	if g.WaitMergeableNodeID == "" || g.ActivityNodeID == "" || g.RunnerNodeID == "" {
		return false
	}

	if !hasCanvasPath(spec.Edges, g.PullRequestTriggerNodeID, g.FindNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.FindNodeID, g.WaitMergeableNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.PushTriggerNodeID, g.ListNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.ListNodeID, g.ForEachNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.ForEachNodeID, g.WaitMergeableNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.WaitMergeableNodeID, g.ActivityNodeID) {
		return false
	}
	if !hasCanvasPath(spec.Edges, g.ActivityNodeID, g.RunnerNodeID) {
		return false
	}

	wait := findIntakeNode(spec.Nodes, g.WaitMergeableNodeID)
	if wait == nil || wait.ComponentName() != prFeedbackWaitMergeableComponent {
		return false
	}
	activity := findIntakeNode(spec.Nodes, g.ActivityNodeID)
	if activity == nil || activity.ComponentName() != prFeedbackActivityComponent {
		return false
	}
	if accessValue, _ := activity.Configuration["access"].(string); accessValue != core.PullRequestActivityAccessExclusive {
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
	if strings.TrimSpace(conflictsBaseBranchFromPushNode(findIntakeNode(spec.Nodes, g.PushTriggerNodeID))) == "" {
		return false
	}

	return true
}

func resolvePRFeedbackGraph(source string, spec models.LiveCanvasSpec) prFeedbackGraph {
	nodes := spec.Nodes
	graph := prFeedbackGraph{
		Source: source,
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
		PushTriggerNodeID: resolveIntakeNode(nodes, prFeedbackPushTriggerNodeID, func(node *models.Node) bool {
			return node.ComponentName() == "github.onPush"
		}),
		ListNodeID: resolveIntakeNode(nodes, prFeedbackListNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackListComponent
		}),
		ForEachNodeID: resolveIntakeNode(nodes, prFeedbackForEachNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackForEachComponent
		}),
		WaitMergeableNodeID: resolveIntakeNode(nodes, prFeedbackWaitMergeableNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackWaitMergeableComponent
		}),
		RunnerNodeID: resolveIntakeNode(nodes, prFeedbackRunnerNodeID, func(node *models.Node) bool {
			return slices.Contains(intakeAnalysisComponents, node.ComponentName())
		}),
	}

	switch {
	case source == models.FactoryPRFeedbackHandlerSourcePullRequestConflicts:
		graph.ActivityNodeID = resolveIntakeNode(nodes, prFeedbackStartConflictRepairNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackActivityComponent
		})
	default:
		graph.ActivityNodeID = resolveIntakeNode(nodes, prFeedbackActivityNodeID, func(node *models.Node) bool {
			return node.ComponentName() == prFeedbackActivityComponent
		})
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
