package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/retry"
)

const (
	WorkflowPayloadType          = "github.workflow.finished"
	WorkflowPassedOutputChannel  = "passed"
	WorkflowFailedOutputChannel  = "failed"
	WorkflowRunStatusCompleted   = "completed"
	WorkflowRunConclusionSuccess = "success"
	WorkflowPollInterval         = 5 * time.Minute
)

type RunWorkflow struct{}

type RunWorkflowExecutionMetadata struct {
	WorkflowRun *WorkflowRunMetadata `json:"workflowRun" mapstructure:"workflowRun"`
}

type WorkflowRunMetadata struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	URL        string `json:"url"`
}

type RunWorkflowSpec struct {
	Repository      string   `json:"repository"`
	WorkflowFile    string   `json:"workflowFile"`
	Ref             string   `json:"ref"`
	Inputs          []Input  `json:"inputs"`
	// AllowedBranches specifies which branches can trigger workflows.
	// If empty and EnforceBranchRestriction is not false, defaults are used.
	AllowedBranches []string `json:"allowedBranches,omitempty"`
	// EnforceBranchRestriction controls branch restriction behavior:
	//   - nil (not set): Use secure defaults (main, master, release, production, staging)
	//   - true: Enforce branch restrictions with AllowedBranches or defaults
	//   - false: Disable branch restrictions (allow any branch) - USE WITH CAUTION
	EnforceBranchRestriction *bool `json:"enforceBranchRestriction,omitempty"`
}

// Default allowed branches (protected branches)
var defaultAllowedBranches = []string{
	"main",
	"master",
	"release",
	"production",
	"staging",
}

type Input struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (r *RunWorkflow) Name() string {
	return "github.runWorkflow"
}

func (r *RunWorkflow) Label() string {
	return "Run Workflow"
}

func (r *RunWorkflow) Description() string {
	return "Run GitHub Actions workflow"
}

func (r *RunWorkflow) Documentation() string {
	return `The Run Workflow component triggers a GitHub Actions workflow and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds and deployments from SuperPlane workflows
- **Automated testing**: Run test suites as part of workflow automation
- **Multi-stage pipelines**: Coordinate complex deployment pipelines
- **Workflow chaining**: Chain multiple GitHub Actions workflows together

## How It Works

1. Dispatches the specified GitHub Actions workflow with optional inputs
2. Waits for the workflow run to complete (monitored via webhook and polling)
3. Routes execution based on workflow conclusion:
   - **Passed channel**: Workflow completed successfully
   - **Failed channel**: Workflow failed or was cancelled

## Configuration

- **Repository**: Select the GitHub repository containing the workflow
- **Workflow File**: Path to the workflow file (e.g., ` + "`.github/workflows/ci.yml`" + `)
- **Branch or Tag**: Git reference to run the workflow on (default: main)
- **Inputs**: Optional workflow inputs as key-value pairs (supports expressions)

## Output Channels

- **Passed**: Emitted when workflow completes successfully
- **Failed**: Emitted when workflow fails or is cancelled

## Notes

- The component automatically sets up webhook monitoring for workflow completion
- Falls back to polling if webhook doesn't arrive
- Can be cancelled, which will cancel the running GitHub Actions workflow`
}

func (r *RunWorkflow) Icon() string {
	return "workflow"
}

func (r *RunWorkflow) Color() string {
	return "gray"
}

func (r *RunWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  WorkflowPassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  WorkflowFailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (r *RunWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "workflowFile",
			Label:    "Workflow file",
			Type:     configuration.FieldTypeString,
			Required: true,
			Default:  ".github/workflows/ci.yml",
		},
		{
			Name:     "ref",
			Label:    "Branch or tag",
			Type:     configuration.FieldTypeGitRef,
			Required: true,
			Default:  "main",
		},
		{
			Name:  "inputs",
			Label: "Inputs",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Input",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		// Security configuration for branch restrictions
		{
			Name:        "allowedBranches",
			Label:       "Allowed Branches",
			Type:        configuration.FieldTypeList,
			Description: "List of branch names or patterns (e.g., main, release/*) allowed for workflow execution. Defaults to main, master, release, production, staging.",
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Branch",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (r *RunWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunWorkflow) Setup(ctx core.SetupContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
	if err != nil {
		return err
	}

	spec := RunWorkflowSpec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	// Request webhook for workflow_run events
	ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType:  "workflow_run",
		Repository: spec.Repository,
	})

	return nil
}

func (r *RunWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate branch restriction for security
	if err := r.validateBranchRestriction(spec); err != nil {
		ctx.Logger.Warnf("Branch restriction validation failed: %v", err)
		return ctx.ExecutionState.Fail("branch_restricted", err.Error())
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	//
	// Dispatch the workflow
	// Make sure it works if user specifies full path,
	// or just the path accepted by the API.
	//
	workflowFile := strings.Replace(spec.WorkflowFile, ".github/workflows/", "", 1)
	_, err = client.Actions.CreateWorkflowDispatchEventByFileName(
		context.Background(),
		appMetadata.Owner,
		spec.Repository,
		workflowFile,
		github.CreateWorkflowDispatchEventRequest{
			Ref:    spec.Ref,
			Inputs: r.buildInputs(ctx, spec.Inputs),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to dispatch workflow: %w", err)
	}

	ctx.Logger.Infof("Workflow dispatched - repository=%s, workflow=%s, ref=%s", spec.Repository, spec.WorkflowFile, spec.Ref)

	//
	// The GitHub API does not return a run ID, so we need to find it.
	// See: https://github.com/orgs/community/discussions/9752
	//
	var run *github.WorkflowRun
	err = retry.WithConstantWait(func() error {
		var findErr error
		run, findErr = r.findWorkflowRun(client, appMetadata.Owner, spec.Repository, ctx.ID.String())
		return findErr
	}, retry.Options{
		Task:         "find workflow run",
		MaxAttempts:  15,
		Wait:         2 * time.Second,
		InitialDelay: time.Second,
		Verbose:      true,
	})

	if err != nil {
		return fmt.Errorf("failed to find workflow run: %w", err)
	}

	// Save workflow run to metadata
	err = ctx.Metadata.Set(RunWorkflowExecutionMetadata{
		WorkflowRun: &WorkflowRunMetadata{
			ID:         run.GetID(),
			Status:     run.GetStatus(),
			Conclusion: run.GetConclusion(),
			URL:        run.GetHTMLURL(),
		},
	})

	if err != nil {
		return err
	}

	// Store workflow run ID in KV for webhook matching
	err = ctx.ExecutionState.SetKV("workflow_run_id", fmt.Sprintf("%d", run.GetID()))
	if err != nil {
		return err
	}

	ctx.Logger.Infof("Started workflow run %d", run.GetID())

	// Schedule poll to check workflow status updates (in case webhook doesn't arrive)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, WorkflowPollInterval)
}

func (r *RunWorkflow) Cancel(ctx core.ExecutionContext) error {
	//
	// Parse metadata and configuration
	//
	metadata := RunWorkflowExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	spec := RunWorkflowSpec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	// If no workflow run ID, nothing to cancel
	if metadata.WorkflowRun == nil || metadata.WorkflowRun.ID == 0 {
		ctx.Logger.Info("No workflow run to cancel")
		return nil
	}

	// If workflow already completed, nothing to cancel
	if metadata.WorkflowRun.Status == WorkflowRunStatusCompleted {
		ctx.Logger.Info("Workflow run already completed, nothing to cancel")
		return nil
	}

	//
	// Create GitHub client, and cancel workflow run
	//
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	response, err := client.Actions.CancelWorkflowRunByID(
		context.Background(),
		appMetadata.Owner,
		spec.Repository,
		metadata.WorkflowRun.ID,
	)

	//
	// GitHub SDK returns an error even though it got a 202 response back :)
	//
	if response.StatusCode == http.StatusAccepted {
		ctx.Logger.Infof("Workflow run %d cancelled", metadata.WorkflowRun.ID)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to cancel workflow run: %w", err)
	}

	return fmt.Errorf(
		"Cancel request for %d received status code %d: %v",
		metadata.WorkflowRun.ID,
		response.StatusCode,
		err,
	)
}

func (r *RunWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	statusCode, err := verifySignature(ctx)
	if err != nil {
		return statusCode, err
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "workflow_run" {
		return http.StatusOK, nil
	}

	// If statusCode is 200 but not the right event type, just ignore
	if statusCode == http.StatusOK {
		eventType := ctx.Headers.Get("X-GitHub-Event")
		if eventType != "workflow_run" {
			return http.StatusOK, nil
		}
	}

	// Parse the entire webhook payload
	var payload map[string]any
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// We only care about completed workflow runs
	action, ok := payload["action"].(string)
	if !ok || action != "completed" {
		return http.StatusOK, nil
	}

	newMetadata, data, err := metadataFromPayload(payload)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error determing new metadata: %v", err)
	}

	//
	// Find the execution associated with this workflow run
	// If an error is returned, it means this run wasn't started by SuperPlane,
	// so we just ignore it.
	//
	executionCtx, err := ctx.FindExecutionByKV("workflow_run_id", fmt.Sprintf("%d", newMetadata.WorkflowRun.ID))
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	// Already finished, do not do anything
	if metadata.WorkflowRun != nil && metadata.WorkflowRun.Status == WorkflowRunStatusCompleted {
		return http.StatusOK, nil
	}

	err = executionCtx.Metadata.Set(newMetadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	if newMetadata.WorkflowRun.Conclusion == WorkflowRunConclusionSuccess {
		err = executionCtx.ExecutionState.Emit(WorkflowPassedOutputChannel, WorkflowPayloadType, []any{data})
	} else {
		err = executionCtx.ExecutionState.Emit(WorkflowFailedOutputChannel, WorkflowPayloadType, []any{data})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func metadataFromPayload(payload map[string]any) (*RunWorkflowExecutionMetadata, map[string]any, error) {
	workflowRun, ok := payload["workflow_run"].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("workflow_run not found in payload")
	}

	runID, ok := workflowRun["id"].(float64)
	if !ok {
		return nil, nil, fmt.Errorf("run ID not found or invalid")
	}

	status, ok := workflowRun["status"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("run status not found or invalid")
	}

	conclusion, ok := workflowRun["conclusion"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("run conclusion not found or invalid")
	}

	htmlURL, ok := workflowRun["html_url"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("run URL not found or invalid")
	}

	return &RunWorkflowExecutionMetadata{
		WorkflowRun: &WorkflowRunMetadata{
			ID:         int64(runID),
			Status:     status,
			Conclusion: conclusion,
			URL:        htmlURL,
		},
	}, workflowRun, nil
}

func (r *RunWorkflow) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (r *RunWorkflow) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *RunWorkflow) poll(ctx core.ActionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If workflow already finished, nothing to do
	if metadata.WorkflowRun.Status == WorkflowRunStatusCompleted {
		return nil
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return err
	}

	// Get the latest status of the workflow run
	run, _, err := client.Actions.GetWorkflowRunByID(
		context.Background(),
		appMetadata.Owner,
		spec.Repository,
		metadata.WorkflowRun.ID,
	)
	if err != nil {
		return err
	}

	// If not finished, poll again
	if run.GetStatus() != WorkflowRunStatusCompleted {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, WorkflowPollInterval)
	}

	// Update metadata with final status
	metadata.WorkflowRun.Status = run.GetStatus()
	metadata.WorkflowRun.Conclusion = run.GetConclusion()
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	// Emit based on conclusion
	if run.GetConclusion() == WorkflowRunConclusionSuccess {
		return ctx.ExecutionState.Emit(WorkflowPassedOutputChannel, WorkflowPayloadType, []any{run})
	}

	return ctx.ExecutionState.Emit(WorkflowFailedOutputChannel, WorkflowPayloadType, []any{run})
}

func (r *RunWorkflow) findWorkflowRun(client *github.Client, owner, repo, executionID string) (*github.WorkflowRun, error) {
	// List recent workflow runs
	runs, _, err := client.Actions.ListRepositoryWorkflowRuns(
		context.Background(),
		owner,
		repo,
		&github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// Find the run with our execution ID in the name
	for _, run := range runs.WorkflowRuns {
		if strings.Contains(run.GetName(), executionID) {
			return run, nil
		}
	}

	return nil, fmt.Errorf("workflow run with execution ID %s not found", executionID)
}

func (r *RunWorkflow) buildInputs(ctx core.ExecutionContext, inputs []Input) map[string]any {
	result := make(map[string]any)

	// Copy user-provided inputs
	for _, input := range inputs {
		result[input.Name] = input.Value
	}

	// Add SuperPlane metadata
	result["superplane_canvas_id"] = ctx.WorkflowID
	result["superplane_execution_id"] = ctx.ID

	return result
}

// validateBranchRestriction checks if the ref is an allowed branch.
// This prevents triggering workflows on unreviewed branches/PRs which could contain malicious code.
func (r *RunWorkflow) validateBranchRestriction(spec RunWorkflowSpec) error {
	// If EnforceBranchRestriction is explicitly set to false, skip all validation
	// This allows users to opt out of branch restrictions when needed
	if spec.EnforceBranchRestriction != nil && !*spec.EnforceBranchRestriction {
		return nil
	}

	// Determine allowed branches:
	// 1. Use user-specified AllowedBranches if provided
	// 2. Otherwise use secure defaults
	allowedBranches := spec.AllowedBranches
	if len(allowedBranches) == 0 {
		allowedBranches = defaultAllowedBranches
	}

	ref := spec.Ref

	// Check if ref is a commit SHA (40 hex characters)
	// Commit SHAs are blocked as they could reference unreviewed code
	if len(ref) == 40 && isHexString(ref) {
		return fmt.Errorf("direct commit SHA references are not allowed for security reasons; use a branch name instead")
	}

	// Check if ref looks like a PR reference (e.g., refs/pull/123/head)
	if strings.HasPrefix(ref, "refs/pull/") || strings.HasPrefix(ref, "pull/") {
		return fmt.Errorf("pull request references are not allowed for security reasons; use a protected branch name instead")
	}

	// Check if ref is in the allowed branches list
	for _, allowed := range allowedBranches {
		// Exact match
		if ref == allowed {
			return nil
		}
		// Match with refs/heads/ prefix
		if ref == "refs/heads/"+allowed {
			return nil
		}
		// Wildcard matching for patterns like "release/*"
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(ref, prefix) || strings.HasPrefix(ref, "refs/heads/"+prefix) {
				return nil
			}
		}
	}

	return fmt.Errorf("ref '%s' is not in the allowed branches list; allowed branches: %v", ref, allowedBranches)
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
