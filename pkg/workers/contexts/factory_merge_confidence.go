package contexts

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
)

// Assessment completion exposes immutable report data to ordinary canvas expressions.
// Provider access and Markdown formatting belong to downstream components and templates.
func (s *ExecutionStateContext) prepareMergeConfidenceOutput(channel, payloadType string, payloads []any) ([]any, error) {
	component := strings.TrimSuffix(payloadType, ".finished")
	if (channel != "passed" && channel != "failed") || payloadType == component || !models.IsAgentRunnerComponent(component) || len(payloads) == 0 || s.IsFinished() || s.IsCancelling() {
		return payloads, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(s.tx, s.execution.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil {
		return payloads, nil
	}
	run, err := models.FindCanvasRunInTransaction(s.tx, canvas.ID, s.execution.RunID)
	if err != nil {
		return nil, err
	}
	version, err := models.FindCanvasVersionInTransaction(s.tx, run.WorkflowID, run.VersionID)
	if err != nil {
		return nil, err
	}
	var source *models.Node
	for i := range version.Nodes {
		if version.Nodes[i].ID == s.execution.NodeID {
			source = &version.Nodes[i]
			break
		}
	}
	if source == nil || source.ComponentName() != component {
		return payloads, nil
	}
	if frozen := s.execution.FrozenComponentName(); frozen != "" && frozen != component {
		return payloads, nil
	}
	assessment, err := mergeConfidenceAssessmentFromNode(*source)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return payloads, nil
	}
	workspace, err := models.FindFactory(s.tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	activity, err := models.FindPullRequestActivityByRunID(s.tx, run.ID)
	if err != nil {
		return nil, err
	}
	pr, err := workspace.FindPullRequest(s.tx, models.FactoryPullRequestLookup{ID: activity.PullRequestID})
	if err != nil {
		return nil, err
	}
	order, err := workspace.FindWorkOrder(s.tx, pr.WorkOrderID)
	if err != nil {
		return nil, err
	}
	assessment.Checks, err = order.ChecksForRun(s.tx, run.ID)
	if err != nil {
		return nil, err
	}
	event, err := run.RootEvent(s.tx)
	if err != nil {
		return nil, err
	}
	assessment.ReviewedCommit = mergeConfidenceReviewedCommit(event)
	if assessment.ReviewedCommit == "" {
		return nil, fmt.Errorf("assessment has no reviewed commit")
	}
	assessment.Passed = channel == "passed" && len(run.Errors) == 0
	assessment.TaskURL = strings.TrimRight(uiBaseURL(), "/") + order.URLPath(workspace.RouteSegment())
	report := assessment.ReportData()
	report.PullRequestID = pr.ID.String()
	if !report.Complete {
		// Publishing the report must not turn an incomplete assessment into a passed run.
		message := "Merge confidence assessment failed."
		var missing []string
		for _, check := range report.Checks {
			if check.Missing {
				missing = append(missing, check.Label)
			}
		}
		if len(missing) > 0 {
			message = "Merge confidence did not report these checks: " + strings.Join(missing, ", ") + ". Run merge confidence again."
		}
		if err := run.AddError(s.tx, message, config.MaxPayloadSize()); err != nil {
			return nil, err
		}
	}
	outputs := make([]any, len(payloads))
	for i, payload := range payloads {
		data, ok := payload.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("assessment completion must emit an object")
		}
		cloned := maps.Clone(data)
		cloned["mergeConfidence"] = report
		outputs[i] = cloned
	}
	return outputs, nil
}

func mergeConfidenceAssessmentFromNode(node models.Node) (*mergeconfidence.Assessment, error) {
	encoded, err := json.Marshal(node.Configuration)
	if err != nil {
		return nil, err
	}
	var configuration struct {
		Environment []struct {
			Name string `json:"name"`
		} `json:"environment"`
		Steps []runner.AgentStep `json:"steps"`
	}
	if err := json.Unmarshal(encoded, &configuration); err != nil {
		return nil, err
	}
	scoped := false
	for _, env := range configuration.Environment {
		if env.Name == runner.EnvSuperplaneMergeConfidenceOrderID {
			scoped = true
			break
		}
	}
	if !scoped {
		return nil, nil
	}
	enabled, labels := runner.MergeConfidenceChecksFromSteps(configuration.Steps)
	return &mergeconfidence.Assessment{Enabled: enabled, Labels: labels}, nil
}

func mergeConfidenceReviewedCommit(event *models.CanvasEvent) string {
	// Bitbucket trigger events also carry this normalized pull_request shape.
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return ""
	}
	pr, _ := payload["pull_request"].(map[string]any)
	head, _ := pr["head"].(map[string]any)
	sha, _ := head["sha"].(string)
	return strings.TrimSpace(sha)
}
