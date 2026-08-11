package components

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/common"
)

const (
	DefaultWaitTimeoutSeconds   = 3600
	DefaultLookupTimeoutSeconds = 60

	// LookupPollInterval is how often we retry looking up a pipeline
	// that hasn't been found yet, either by ID or by branch+commit.
	LookupPollInterval = 10 * time.Second

	waitForPipelineHookLookup        = "lookup"
	waitForPipelineHookLookupTimeout = "lookupTimeout"
	waitForPipelineHookPoll          = "poll"
	waitForPipelineHookWaitTimeout   = "waitTimeout"
)

type WaitForPipeline struct{}

type WaitForPipelineNodeMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

// WaitForPipelineExecutionMetadata tracks the state of a single execution.
//
// ProjectID is stored here (rather than only relying on node metadata) because
// internal hooks (core.ActionHookContext) don't have access to node metadata,
// only to the execution's own metadata and to the node's raw configuration.
type WaitForPipelineExecutionMetadata struct {
	ProjectID string            `json:"projectId,omitempty" mapstructure:"projectId,omitempty"`
	Workflow  *WorkflowMetadata `json:"workflow,omitempty" mapstructure:"workflow,omitempty"`
	Pipeline  *PipelineMetadata `json:"pipeline,omitempty" mapstructure:"pipeline,omitempty"`
}

type WaitForPipelineSpec struct {
	Project              string `json:"project" mapstructure:"project"`
	PipelineID           string `json:"pipelineId" mapstructure:"pipelineId"`
	Branch               string `json:"branch" mapstructure:"branch"`
	CommitSha            string `json:"commitSha" mapstructure:"commitSha"`
	TimeoutSeconds       int    `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	LookupTimeoutSeconds int    `json:"lookupTimeoutSeconds" mapstructure:"lookupTimeoutSeconds"`
}

func (w *WaitForPipeline) Name() string {
	return "semaphore.waitForPipeline"
}

func (w *WaitForPipeline) Label() string {
	return "Wait for Pipeline"
}

func (w *WaitForPipeline) Description() string {
	return "Wait for a Semaphore pipeline to finish"
}

func (w *WaitForPipeline) Documentation() string {
	return `The Wait for Pipeline component waits for a Semaphore pipeline that was started elsewhere to finish, and routes the flow based on its result.

Unlike ` + "`semaphore.runWorkflow`" + `, this component does not start a pipeline. It is meant to be used when a pipeline is triggered by something outside of SuperPlane (another CI trigger, the Semaphore UI, a Git push, etc.) and you just need to know when it's done.

## Use Cases

- **Coordinate with external triggers**: Wait for a pipeline that was triggered by a Git push or by the Semaphore UI
- **Fan-in**: Wait for a specific commit's pipeline to finish before continuing a SuperPlane flow
- **Decoupled orchestration**: Watch a pipeline without owning its lifecycle

## How It Works

1. Resolves the pipeline to wait for, either directly by ID, or by looking up the most recent pipeline for a given branch and commit SHA
2. If the pipeline can't be found yet, retries for up to **Lookup Timeout** seconds
3. Once found, waits for the pipeline to complete (monitored via webhook and polling), for up to **Timeout** seconds
4. Routes execution based on the pipeline result:
   - **Passed channel**: Pipeline completed successfully
   - **Failed channel**: Pipeline failed or was cancelled

## Configuration

- **Project**: Select the Semaphore project the pipeline belongs to
- **Pipeline ID**: Reference a pipeline directly by its ID
- **Branch** and **Commit SHA**: Alternatively, look up the pipeline by branch and commit SHA (both must be set)
- **Timeout (seconds)**: How long to wait for the pipeline to finish once it has been found (default 3600)
- **Lookup Timeout (seconds)**: How long to keep trying to find the pipeline before giving up (default 60)

Exactly one of **Pipeline ID** or (**Branch** and **Commit SHA**) must be provided.

## Output Channels

- **Passed**: Emitted when the pipeline completes successfully
- **Failed**: Emitted when the pipeline fails or is cancelled

## Notes

- The component automatically sets up webhook monitoring for pipeline completion
- Falls back to polling if the webhook doesn't arrive
- The component never starts or cancels the Semaphore pipeline - it only observes it`
}

func (w *WaitForPipeline) Icon() string {
	return "workflow"
}

func (w *WaitForPipeline) Color() string {
	return "gray"
}

func (w *WaitForPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  PassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (w *WaitForPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Description: "Reference the pipeline directly by ID. Either this, or both Branch and Commit SHA, must be provided.",
			Placeholder: "e.g. {{ event.pipeline.id }}",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Used together with Commit SHA to look up the pipeline.",
		},
		{
			Name:        "commitSha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Description: "Used together with Branch to look up the pipeline.",
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Default:     DefaultWaitTimeoutSeconds,
			Description: "Maximum time to wait for the pipeline to finish once it has been found. The run fails if the pipeline is still not done after this many seconds.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
				},
			},
		},
		{
			Name:        "lookupTimeoutSeconds",
			Label:       "Lookup Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Default:     DefaultLookupTimeoutSeconds,
			Description: "Maximum time to spend trying to find the pipeline before failing the run.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
				},
			},
		},
	}
}

func (w *WaitForPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (w *WaitForPipeline) Setup(ctx core.SetupContext) error {
	spec, err := decodeWaitForPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateWaitForPipelineSpec(spec); err != nil {
		return err
	}

	metadata := WaitForPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && (spec.Project == metadata.Project.ID || spec.Project == metadata.Project.Name) {
		return nil
	}

	client, err := common.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", spec.Project, err)
	}

	err = ctx.Metadata.Set(WaitForPipelineNodeMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		Project: project.Metadata.ProjectName,
	})

	return nil
}

func (w *WaitForPipeline) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeWaitForPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateWaitForPipelineSpec(spec); err != nil {
		return err
	}

	nodeMetadata := WaitForPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if nodeMetadata.Project == nil {
		return fmt.Errorf("project not resolved yet")
	}

	client, err := common.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := findPipeline(client, spec, nodeMetadata.Project.ID)
	if err != nil {
		return err
	}

	//
	// Pipeline not found yet. Store what we need to keep retrying, and
	// schedule the lookup retry loop and its overall deadline.
	//
	if pipeline == nil {
		err = ctx.Metadata.Set(WaitForPipelineExecutionMetadata{
			ProjectID: nodeMetadata.Project.ID,
		})
		if err != nil {
			return err
		}

		err = ctx.Requests.ScheduleActionCall(waitForPipelineHookLookup, map[string]any{}, LookupPollInterval)
		if err != nil {
			return err
		}

		return ctx.Requests.ScheduleActionCall(
			waitForPipelineHookLookupTimeout,
			map[string]any{},
			time.Duration(spec.LookupTimeoutSeconds)*time.Second,
		)
	}

	return handlePipelineFound(ctx.ExecutionState, ctx.Metadata, ctx.Requests, client.OrgURL, pipeline, spec)
}

func (w *WaitForPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (w *WaitForPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	signature := ctx.Headers.Get("X-Semaphore-Signature-256")
	if signature == "" {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature")
	}

	var payload map[string]any
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	pipelineData, ok := payload["pipeline"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, nil, fmt.Errorf("pipeline data missing from webhook payload")
	}

	pipelineID, _ := pipelineData["id"].(string)
	if pipelineID == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("pipeline id missing from webhook payload")
	}

	pipelineState, _ := pipelineData["state"].(string)
	pipelineResult, _ := pipelineData["result"].(string)

	ctx.Logger.Infof("Received webhook for pipeline %s (state=%s, result=%s)", pipelineID, pipelineState, pipelineResult)

	executionCtx, err := ctx.FindExecutionByKV("pipeline", pipelineID)

	//
	// We will receive hooks for pipelines that this component isn't watching,
	// so we just ignore them.
	//
	if err != nil {
		ctx.Logger.Infof("No execution found for pipeline %s: %v", pipelineID, err)
		return http.StatusOK, nil, nil
	}

	metadata := WaitForPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error decoding metadata: %v", err)
	}

	//
	// Already finished, do not do anything.
	//
	if metadata.Pipeline != nil && metadata.Pipeline.State == PipelineStateDone {
		ctx.Logger.Infof("Pipeline %s already marked as done, skipping", pipelineID)
		return http.StatusOK, nil, nil
	}

	if metadata.Pipeline == nil {
		metadata.Pipeline = &PipelineMetadata{ID: pipelineID}
	}

	metadata.Pipeline.State = pipelineState
	metadata.Pipeline.Result = pipelineResult
	err = executionCtx.Metadata.Set(metadata)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error setting metadata: %v", err)
	}

	if metadata.Workflow != nil && metadata.Workflow.URL != "" {
		workflowData, ok := payload["workflow"].(map[string]any)
		if !ok {
			workflowData = map[string]any{}
			payload["workflow"] = workflowData
		}
		workflowData["url"] = metadata.Workflow.URL
	}

	if metadata.Pipeline.Result == PipelineResultPassed {
		err = executionCtx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{payload})
	} else {
		err = executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	ctx.Logger.Infof("Pipeline %s finished with result=%s, emitted to channel", pipelineID, pipelineResult)
	return http.StatusOK, nil, nil
}

func (w *WaitForPipeline) Hooks() []core.Hook {
	return []core.Hook{
		{Name: waitForPipelineHookLookup, Type: core.HookTypeInternal},
		{Name: waitForPipelineHookLookupTimeout, Type: core.HookTypeInternal},
		{Name: waitForPipelineHookPoll, Type: core.HookTypeInternal},
		{Name: waitForPipelineHookWaitTimeout, Type: core.HookTypeInternal},
	}
}

func (w *WaitForPipeline) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case waitForPipelineHookLookup:
		return w.lookup(ctx)
	case waitForPipelineHookLookupTimeout:
		return w.lookupTimeout(ctx)
	case waitForPipelineHookPoll:
		return w.poll(ctx)
	case waitForPipelineHookWaitTimeout:
		return w.waitTimeout(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (w *WaitForPipeline) lookup(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := WaitForPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	//
	// Already resolved, possibly by a webhook that arrived while we were
	// still retrying the lookup. Nothing else to do.
	//
	if metadata.Pipeline != nil {
		return nil
	}

	spec, err := decodeWaitForPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := findPipeline(client, spec, metadata.ProjectID)
	if err != nil {
		return err
	}

	if pipeline == nil {
		return ctx.Requests.ScheduleActionCall(waitForPipelineHookLookup, map[string]any{}, LookupPollInterval)
	}

	return handlePipelineFound(ctx.ExecutionState, ctx.Metadata, ctx.Requests, client.OrgURL, pipeline, spec)
}

func (w *WaitForPipeline) lookupTimeout(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := WaitForPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	//
	// Already resolved, nothing to do.
	//
	if metadata.Pipeline != nil {
		return nil
	}

	spec, err := decodeWaitForPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Fail(
		"lookupTimeout",
		fmt.Sprintf("could not find pipeline within %ds", spec.LookupTimeoutSeconds),
	)
}

func (w *WaitForPipeline) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := WaitForPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	//
	// If the pipeline already finished, we don't need to do anything.
	//
	if metadata.Pipeline == nil || metadata.Pipeline.State == PipelineStateDone {
		return nil
	}

	client, err := common.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	//
	// If not finished, poll again later.
	//
	if pipeline.State != PipelineStateDone {
		return ctx.Requests.ScheduleActionCall(waitForPipelineHookPoll, map[string]any{}, PollInterval)
	}

	metadata.Pipeline.State = pipeline.State
	metadata.Pipeline.Result = pipeline.Result
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
	}
	if metadata.Workflow != nil && metadata.Workflow.URL != "" {
		payload["workflow"] = map[string]any{
			"url": metadata.Workflow.URL,
		}
	}

	if pipeline.Result == PipelineResultPassed {
		return ctx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
}

func (w *WaitForPipeline) waitTimeout(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := WaitForPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	//
	// Already done, nothing to do.
	//
	if metadata.Pipeline != nil && metadata.Pipeline.State == PipelineStateDone {
		return nil
	}

	spec, err := decodeWaitForPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Fail(
		"timeout",
		fmt.Sprintf("pipeline did not finish within %ds", spec.TimeoutSeconds),
	)
}

func (w *WaitForPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

// findPipeline resolves the pipeline to wait for, either by ID or by
// branch+commit. It returns (nil, nil) when the pipeline could not be
// found yet (the caller should keep retrying), and a non-nil error for
// anything else (auth/HTTP errors, etc.) which callers should treat as fatal.
func findPipeline(client *common.Client, spec WaitForPipelineSpec, projectID string) (*common.Pipeline, error) {
	if spec.PipelineID != "" {
		pipeline, err := client.GetPipeline(spec.PipelineID)
		if err != nil {
			if common.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, err
		}

		return pipeline, nil
	}

	pipeline, err := client.FindPipelineByBranchAndCommit(projectID, spec.Branch, spec.CommitSha)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return pipeline, nil
}

// handlePipelineFound is the shared handling for once a pipeline has been resolved,
// used both by Execute() (when the pipeline is found immediately) and by the lookup
// hook (when it's found on a retry).
func handlePipelineFound(
	executionState core.ExecutionStateContext,
	metadataWriter core.MetadataWriter,
	requests core.RequestContext,
	orgURL string,
	pipeline *common.Pipeline,
	spec WaitForPipelineSpec,
) error {
	var workflow *WorkflowMetadata
	if pipeline.WorkflowID != "" {
		workflow = &WorkflowMetadata{
			ID:  pipeline.WorkflowID,
			URL: fmt.Sprintf("%s/workflows/%s", orgURL, pipeline.WorkflowID),
		}
	}

	metadata := WaitForPipelineExecutionMetadata{
		Workflow: workflow,
		Pipeline: &PipelineMetadata{
			ID:     pipeline.PipelineID,
			State:  pipeline.State,
			Result: pipeline.Result,
		},
	}

	if err := metadataWriter.Set(metadata); err != nil {
		return err
	}

	//
	// This is what allows the component to associate a semaphore webhook
	// for a pipeline finishing to a SuperPlane execution.
	//
	// We do this regardless of whether the pipeline is already done, since a
	// webhook for it could theoretically race with us.
	//
	if err := executionState.SetKV("pipeline", pipeline.PipelineID); err != nil {
		return err
	}

	if pipeline.State == PipelineStateDone {
		payload := map[string]any{
			"pipeline": pipeline,
		}
		if workflow != nil {
			payload["workflow"] = map[string]any{
				"url": workflow.URL,
			}
		}

		if pipeline.Result == PipelineResultPassed {
			return executionState.Emit(PassedOutputChannel, PayloadType, []any{payload})
		}

		return executionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	//
	// We still set up the poller to check for pipeline finishing,
	// just in case something wrong happens with the update through the webhook.
	//
	if err := requests.ScheduleActionCall(waitForPipelineHookPoll, map[string]any{}, PollInterval); err != nil {
		return err
	}

	return requests.ScheduleActionCall(
		waitForPipelineHookWaitTimeout,
		map[string]any{},
		time.Duration(spec.TimeoutSeconds)*time.Second,
	)
}

func decodeWaitForPipelineSpec(raw any) (WaitForPipelineSpec, error) {
	spec := WaitForPipelineSpec{}
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return WaitForPipelineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.TimeoutSeconds == 0 {
		spec.TimeoutSeconds = DefaultWaitTimeoutSeconds
	}
	if spec.LookupTimeoutSeconds == 0 {
		spec.LookupTimeoutSeconds = DefaultLookupTimeoutSeconds
	}

	return spec, nil
}

func validateWaitForPipelineSpec(spec WaitForPipelineSpec) error {
	if spec.Project == "" {
		return fmt.Errorf("project is required")
	}

	usingPipelineID := spec.PipelineID != ""
	usingBranchAndCommit := spec.Branch != "" || spec.CommitSha != ""

	if usingPipelineID && usingBranchAndCommit {
		return fmt.Errorf("either pipelineId or both branch and commitSha must be provided, not both")
	}

	if usingPipelineID {
		return nil
	}

	if spec.Branch == "" || spec.CommitSha == "" {
		return fmt.Errorf("either pipelineId or both branch and commitSha must be provided")
	}

	return nil
}

func intPtr(v int) *int {
	return &v
}
