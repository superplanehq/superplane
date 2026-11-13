package mergeconfidence

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// PrepareOutputs builds immutable completion data before shared rendering and persistence.
// It runs for each completion attempt, after checks have been reported.
func PrepareOutputs(tx *gorm.DB, execution *models.CanvasNodeExecution) (payloadType string, outputs map[string]core.ExecutionOutput, prepareErr error) {
	defer func() {
		if prepareErr != nil {
			outputs = map[string]core.ExecutionOutput{"passed": {}, "failed": {}}
		}
	}()
	component := execution.FrozenComponentName()
	if component != "" {
		payloadType = component + ".finished"
	}
	if (component != "" && !models.IsAgentRunnerComponent(component)) || execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return payloadType, nil, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return payloadType, nil, err
	}
	if canvas.FactoryID == nil {
		return payloadType, nil, nil
	}
	run, err := models.FindCanvasRunInTransaction(tx, canvas.ID, execution.RunID)
	if err != nil {
		return payloadType, nil, err
	}
	version, err := models.FindCanvasVersionInTransaction(tx, run.WorkflowID, run.VersionID)
	if err != nil {
		return payloadType, nil, err
	}
	var source *models.Node
	for i := range version.Nodes {
		if version.Nodes[i].ID == execution.NodeID {
			source = &version.Nodes[i]
			break
		}
	}
	if source != nil && component == "" {
		component = source.ComponentName()
		payloadType = component + ".finished"
	}
	if !models.IsAgentRunnerComponent(component) || source == nil || source.ComponentName() != component {
		return payloadType, nil, nil
	}
	if frozen := execution.FrozenComponentName(); frozen != "" && frozen != component {
		return payloadType, nil, nil
	}
	assessment, err := mergeConfidenceAssessmentFromNode(*source)
	if err != nil {
		return payloadType, nil, err
	}
	if assessment == nil {
		return payloadType, nil, nil
	}
	workspace, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return payloadType, nil, err
	}
	activity, err := models.FindPullRequestActivityByRunID(tx, run.ID)
	if err != nil {
		return payloadType, nil, err
	}
	pr, err := workspace.FindPullRequest(tx, models.FactoryPullRequestLookup{ID: activity.PullRequestID})
	if err != nil {
		return payloadType, nil, err
	}
	order, err := workspace.FindWorkOrder(tx, pr.WorkOrderID)
	if err != nil {
		return payloadType, nil, err
	}
	assessment.Checks, err = order.ChecksForRun(tx, run.ID)
	if err != nil {
		return payloadType, nil, err
	}
	event, err := run.RootEvent(tx)
	if err != nil {
		return payloadType, nil, err
	}
	assessment.ReviewedCommit = mergeConfidenceReviewedCommit(event)
	if assessment.ReviewedCommit == "" {
		return payloadType, nil, fmt.Errorf("assessment has no reviewed commit")
	}
	assessment.TaskURL = strings.TrimRight(baseURL(), "/") + order.URLPath(workspace.RouteSegment())
	outputs = map[string]core.ExecutionOutput{}
	for _, channel := range []string{"passed", "failed"} {
		assessment.Passed = channel == "passed" && len(run.Errors) == 0
		report := assessment.ReportData()
		report.PullRequestID = pr.ID.String()
		report.Repository = pr.Repository
		report.PullNumber = pr.Number
		output := core.ExecutionOutput{Fields: map[string]any{"mergeConfidence": report}}
		if channel == "failed" {
			output.ErrorMessage = "Merge confidence assessment failed."
		}
		if channel == "passed" && !report.Complete {
			var missing []string
			for _, check := range report.Checks {
				if check.Missing {
					missing = append(missing, check.Label)
				}
			}
			output.ErrorMessage = "Merge confidence assessment failed."
			if len(missing) > 0 {
				output.ErrorMessage = "Merge confidence did not report these checks: " + strings.Join(missing, ", ") + ". Run merge confidence again."
			}
		}
		outputs[channel] = output
	}
	return payloadType, outputs, nil
}

func mergeConfidenceAssessmentFromNode(node models.Node) (*Assessment, error) {
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
	enabled, labels := MergeConfidenceChecksFromSteps(configuration.Steps)
	return &Assessment{Enabled: enabled, Labels: labels}, nil
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

func baseURL() string {
	if value := os.Getenv("BASE_URL"); value != "" {
		return value
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}
