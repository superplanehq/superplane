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
	Repository   string  `json:"repository"`
	WorkflowFile string  `json:"workflowFile"`
	Ref          string  `json:"ref"`
	Inputs       []Input `json:"inputs"`
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
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:        "workflowFile",
			Label:       "Workflow file",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. .github/workflows/ci.yml",
		},
		{
			Name:     "ref",
			Label:    "Branch or tag",
			Type:     configuration.FieldTypeGitRef,
			Required: true,
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
								Name:     "name",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
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
	}
}

func (r *RunWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunWorkflow) Setup(ctx core.SetupContext) error {
	err := ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
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
	ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
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

	metadata := NodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallationContext, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	//
	// Make sure it works if user specifies full path,
	// or just the path accepted by the API.
	//
	workflowFile := strings.Replace(spec.WorkflowFile, ".github/workflows/", "", 1)

	// Dispatch the workflow
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
	execMetadata := RunWorkflowExecutionMetadata{
		WorkflowRun: &WorkflowRunMetadata{
			ID:         run.GetID(),
			Status:     run.GetStatus(),
			Conclusion: run.GetConclusion(),
			URL:        run.GetHTMLURL(),
		},
	}
	err = ctx.MetadataContext.Set(execMetadata)
	if err != nil {
		return err
	}

	// Store workflow run ID in KV for webhook matching
	err = ctx.ExecutionStateContext.SetKV("workflow_run_id", fmt.Sprintf("%d", run.GetID()))
	if err != nil {
		return err
	}

	ctx.Logger.Infof("Found workflow run - id=%d, status=%s, url=%s", run.GetID(), run.GetStatus(), run.GetHTMLURL())

	// Schedule poll to check workflow status updates (in case webhook doesn't arrive)
	return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, WorkflowPollInterval)
}

func (r *RunWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	statusCode, err := verifySignature(ctx, "workflow_run")
	if err != nil {
		return statusCode, err
	}

	// If statusCode is 200 but not the right event type, just ignore
	if statusCode == http.StatusOK {
		eventType := ctx.Headers.Get("X-GitHub-Event")
		if eventType != "workflow_run" {
			return http.StatusOK, nil
		}
	}

	type Hook struct {
		Action      string `json:"action"`
		WorkflowRun struct {
			ID         int64  `json:"id"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HTMLURL    string `json:"html_url"`
		} `json:"workflow_run"`
	}

	hook := Hook{}
	err = json.Unmarshal(ctx.Body, &hook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// We only care about completed workflow runs
	if hook.Action != "completed" {
		return http.StatusOK, nil
	}

	// Find the execution associated with this workflow run
	executionCtx, err := ctx.FindExecutionByKV("workflow_run_id", fmt.Sprintf("%d", hook.WorkflowRun.ID))
	if err != nil {
		// This workflow run wasn't started by SuperPlane, ignore it
		return http.StatusOK, nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.MetadataContext.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	// Already finished, do not do anything
	if metadata.WorkflowRun != nil && metadata.WorkflowRun.Status == WorkflowRunStatusCompleted {
		return http.StatusOK, nil
	}

	// Update metadata
	if metadata.WorkflowRun == nil {
		metadata.WorkflowRun = &WorkflowRunMetadata{}
	}
	metadata.WorkflowRun.ID = hook.WorkflowRun.ID
	metadata.WorkflowRun.Status = hook.WorkflowRun.Status
	metadata.WorkflowRun.Conclusion = hook.WorkflowRun.Conclusion
	metadata.WorkflowRun.URL = hook.WorkflowRun.HTMLURL

	err = executionCtx.MetadataContext.Set(metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	// Emit based on conclusion
	if hook.WorkflowRun.Conclusion == WorkflowRunConclusionSuccess {
		err = executionCtx.ExecutionStateContext.Emit(WorkflowPassedOutputChannel, WorkflowPayloadType, []any{metadata})
	} else {
		err = executionCtx.ExecutionStateContext.Emit(WorkflowFailedOutputChannel, WorkflowPayloadType, []any{metadata})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
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

	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If workflow already finished, nothing to do
	if metadata.WorkflowRun.Status == WorkflowRunStatusCompleted {
		return nil
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallationContext, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
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
		return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, WorkflowPollInterval)
	}

	// Update metadata with final status
	metadata.WorkflowRun.Status = run.GetStatus()
	metadata.WorkflowRun.Conclusion = run.GetConclusion()
	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return err
	}

	// Emit based on conclusion
	if run.GetConclusion() == WorkflowRunConclusionSuccess {
		return ctx.ExecutionStateContext.Emit(WorkflowPassedOutputChannel, WorkflowPayloadType, []any{metadata})
	}

	return ctx.ExecutionStateContext.Emit(WorkflowFailedOutputChannel, WorkflowPayloadType, []any{metadata})
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
