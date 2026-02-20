package codepipeline

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
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	PayloadType = "aws.codepipeline.pipeline.finished"

	PassedOutputChannel = "passed"
	FailedOutputChannel = "failed"

	PipelineStatusInProgress = "InProgress"
	PipelineStatusSucceeded  = "Succeeded"
	PipelineStatusFailed     = "Failed"
	PipelineStatusStopped    = "Stopped"
	PipelineStatusStopping   = "Stopping"

	PollInterval = 5 * time.Minute
)

type RunPipeline struct{}

type RunPipelineSpec struct {
	Region   string `json:"region" mapstructure:"region"`
	Pipeline string `json:"pipeline" mapstructure:"pipeline"`
}

// RunPipelineNodeMetadata is cached during Setup() to avoid repeated API calls.
type RunPipelineNodeMetadata struct {
	Region         string            `json:"region,omitempty" mapstructure:"region,omitempty"`
	Pipeline       *PipelineMetadata `json:"pipeline" mapstructure:"pipeline"`
	SubscriptionID string            `json:"subscriptionId,omitempty" mapstructure:"subscriptionId,omitempty"`
}

type PipelineMetadata struct {
	Name string `json:"name"`
}

// RunPipelineExecutionMetadata tracks per-execution state.
type RunPipelineExecutionMetadata struct {
	Pipeline  *PipelineMetadata  `json:"pipeline" mapstructure:"pipeline"`
	Execution *ExecutionMetadata `json:"execution" mapstructure:"execution"`
	Extra     map[string]any     `json:"extra,omitempty" mapstructure:"extra,omitempty"`
}

type ExecutionMetadata struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// terminalStatusFromEventBridgeState maps an EventBridge pipeline execution
// state (uppercase, e.g. "SUCCEEDED") to the CodePipeline API status constant.
// It returns ("", false) for non-terminal states such as STARTED or RESUMED.
func terminalStatusFromEventBridgeState(state string) (string, bool) {
	switch state {
	case "SUCCEEDED":
		return PipelineStatusSucceeded, true
	case "FAILED":
		return PipelineStatusFailed, true
	case "CANCELED", "CANCELLED", "STOPPED":
		return PipelineStatusStopped, true
	default:
		return "", false
	}
}

func eventBridgeStateFromPipelineStatus(status string) string {
	switch status {
	case PipelineStatusSucceeded:
		return "SUCCEEDED"
	case PipelineStatusFailed:
		return "FAILED"
	case PipelineStatusStopped:
		return "STOPPED"
	default:
		return strings.ToUpper(status)
	}
}

func pipelineExecutionOutputPayload(pipelineName, executionID, status, state string, detail map[string]any) map[string]any {
	return map[string]any{
		"pipeline": map[string]any{
			"name":        pipelineName,
			"executionId": executionID,
			"status":      status,
			"state":       state,
		},
		"detail": detail,
	}
}

func (r *RunPipeline) Name() string {
	return "aws.codepipeline.runPipeline"
}

func (r *RunPipeline) Label() string {
	return "CodePipeline • Run Pipeline"
}

func (r *RunPipeline) Description() string {
	return "Start an AWS CodePipeline execution and wait for it to complete"
}

func (r *RunPipeline) Documentation() string {
	return `The Run Pipeline component triggers an AWS CodePipeline execution and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger deployments from SuperPlane workflows
- **Pipeline automation**: Run CodePipeline pipelines as part of workflow automation
- **Multi-stage deployments**: Coordinate complex deployment pipelines
- **Workflow chaining**: Chain multiple CodePipeline pipelines together

## How It Works

1. Starts a CodePipeline execution with the specified pipeline name
2. Waits for the pipeline to complete (monitored via EventBridge webhook and polling)
3. Routes execution based on pipeline result:
   - **Passed channel**: Pipeline completed successfully
   - **Failed channel**: Pipeline failed or was cancelled

## Configuration

- **Region**: AWS region where the pipeline exists
- **Pipeline**: Pipeline name or ARN to execute

## Output Channels

- **Passed**: Emitted when pipeline completes successfully
- **Failed**: Emitted when pipeline fails or is cancelled

## Notes

- The component automatically sets up EventBridge monitoring for pipeline completion
- Falls back to polling if webhook doesn't arrive
- Can be cancelled, which will stop the running pipeline execution`
}

func (r *RunPipeline) Icon() string {
	return "aws"
}

func (r *RunPipeline) Color() string {
	return "orange"
}

func (r *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
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

func (r *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "pipeline",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "CodePipeline pipeline to execute",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codepipeline.pipeline",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (r *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunPipeline) Setup(ctx core.SetupContext) error {
	spec := RunPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Region == "" {
		return fmt.Errorf("region is required")
	}
	if spec.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}

	metadata := RunPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		metadata = RunPipelineNodeMetadata{}
	}

	if metadata.SubscriptionID != "" && metadata.Pipeline != nil && spec.Pipeline == metadata.Pipeline.Name && spec.Region == metadata.Region {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	pipelines, err := client.ListPipelines()
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	var foundPipeline *PipelineMetadata
	for _, p := range pipelines {
		if p.Name == spec.Pipeline {
			foundPipeline = &PipelineMetadata{
				Name: p.Name,
			}
			break
		}
	}

	if foundPipeline == nil {
		return fmt.Errorf("pipeline not found: %s", spec.Pipeline)
	}

	// Provision EventBridge rule if not already present for CodePipeline events.
	source := "aws.codepipeline"
	detailType := "CodePipeline Pipeline Execution State Change"

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, source, spec.Region, detailType)
	if err != nil {
		ctx.Logger.Warnf("Failed to check EventBridge rule availability: %v", err)
	}

	if !hasRule {
		err = ctx.Integration.ScheduleActionCall(
			"provisionRule",
			common.ProvisionRuleParameters{
				Region:     spec.Region,
				Source:     source,
				DetailType: detailType,
			},
			time.Second,
		)
		if err != nil {
			ctx.Logger.Warnf("Failed to schedule EventBridge rule provisioning: %v", err)
		}
	}

	subscriptionID, err := ctx.Integration.Subscribe(&common.EventBridgeEvent{
		Region:     spec.Region,
		DetailType: "CodePipeline Pipeline Execution State Change",
		Source:     "aws.codepipeline",
	})

	nodeMetadata := RunPipelineNodeMetadata{
		Region:   spec.Region,
		Pipeline: foundPipeline,
	}

	if err != nil {
		ctx.Logger.Warnf("Failed to subscribe to CodePipeline events: %v", err)
	} else {
		nodeMetadata.SubscriptionID = subscriptionID.String()
	}

	err = ctx.Metadata.Set(nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (r *RunPipeline) Execute(ctx core.ExecutionContext) error {
	spec := RunPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	nodeMetadata := RunPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Pipeline == nil {
		return fmt.Errorf("pipeline metadata not found - component may not be properly set up")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	response, err := client.StartPipelineExecution(nodeMetadata.Pipeline.Name)
	if err != nil {
		return fmt.Errorf("failed to start pipeline execution: %w", err)
	}

	executionID := response.PipelineExecutionID
	ctx.Logger.Infof("Started pipeline execution - pipeline=%s, execution=%s", nodeMetadata.Pipeline.Name, executionID)

	err = ctx.Metadata.Set(RunPipelineExecutionMetadata{
		Pipeline: nodeMetadata.Pipeline,
		Execution: &ExecutionMetadata{
			ID:     executionID,
			Status: PipelineStatusInProgress,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	// Store execution ID in KV so HandleWebhook can match EventBridge events to this execution.
	err = ctx.ExecutionState.SetKV("pipeline_execution_id", executionID)
	if err != nil {
		return fmt.Errorf("failed to set execution ID: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	metadata := RunPipelineExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Execution == nil || metadata.Execution.ID == "" {
		return nil
	}

	if metadata.Pipeline == nil {
		return nil
	}

	if metadata.Execution.Status == PipelineStatusSucceeded ||
		metadata.Execution.Status == PipelineStatusFailed ||
		metadata.Execution.Status == PipelineStatusStopped {
		return nil
	}

	spec := RunPipelineSpec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Failed to get AWS credentials for cancellation: %v", err)
		return nil
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)
	err = client.StopPipelineExecution(
		metadata.Pipeline.Name,
		metadata.Execution.ID,
		"Cancelled by SuperPlane workflow",
		true,
	)
	if err != nil {
		ctx.Logger.Warnf("Failed to stop pipeline execution: %v", err)
		return nil
	}

	ctx.Logger.Infof("Stopped pipeline execution %s", metadata.Execution.ID)
	return nil
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var payload map[string]any
	err := json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	detail, ok := payload["detail"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing detail in event payload")
	}

	executionID, ok := detail["execution-id"].(string)
	if !ok || executionID == "" {
		return http.StatusBadRequest, fmt.Errorf("missing execution-id in event detail")
	}

	state, ok := detail["state"].(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing state in event detail")
	}

	executionCtx, err := ctx.FindExecutionByKV("pipeline_execution_id", executionID)
	if err != nil {
		return http.StatusOK, nil
	}

	if executionCtx == nil {
		return http.StatusOK, nil
	}

	metadata := RunPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	if metadata.Execution == nil {
		return http.StatusOK, nil
	}

	if metadata.Pipeline == nil {
		return http.StatusOK, nil
	}

	if metadata.Execution.Status == PipelineStatusSucceeded || metadata.Execution.Status == PipelineStatusFailed || metadata.Execution.Status == PipelineStatusStopped {
		return http.StatusOK, nil
	}

	status, terminal := terminalStatusFromEventBridgeState(state)
	if !terminal {
		return http.StatusOK, nil
	}

	metadata.Execution.Status = status
	err = executionCtx.Metadata.Set(metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	outputPayload := pipelineExecutionOutputPayload(metadata.Pipeline.Name, executionID, status, state, detail)

	if status == PipelineStatusSucceeded {
		err = executionCtx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{outputPayload})
	} else {
		err = executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{outputPayload})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
			Description:    "Check pipeline execution status",
		},
		{
			Name:           "finish",
			UserAccessible: true,
			Description:    "Manually finish the execution",
			Parameters: []configuration.Field{
				{
					Name:     "data",
					Type:     configuration.FieldTypeObject,
					Required: false,
					Default:  map[string]any{},
				},
			},
		},
	}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	case "finish":
		return r.finish(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (r *RunPipeline) poll(ctx core.ActionContext) error {
	spec := RunPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunPipelineExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Pipeline == nil {
		return fmt.Errorf("pipeline metadata not found - component may not be properly set up")
	}

	if metadata.Execution == nil {
		return fmt.Errorf("execution metadata not found - component may not have started properly")
	}

	if metadata.Execution.Status == PipelineStatusSucceeded || metadata.Execution.Status == PipelineStatusFailed || metadata.Execution.Status == PipelineStatusStopped {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	execution, err := client.GetPipelineExecution(metadata.Pipeline.Name, metadata.Execution.ID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline execution: %w", err)
	}

	if execution.Status == PipelineStatusInProgress || execution.Status == PipelineStatusStopping {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	metadata.Execution.Status = execution.Status
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	state := eventBridgeStateFromPipelineStatus(execution.Status)
	detail := map[string]any{
		"pipeline":     metadata.Pipeline.Name,
		"execution-id": execution.PipelineExecutionID,
		"state":        state,
	}

	outputPayload := pipelineExecutionOutputPayload(
		metadata.Pipeline.Name,
		execution.PipelineExecutionID,
		execution.Status,
		state,
		detail,
	)

	if execution.Status == PipelineStatusSucceeded {
		return ctx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{outputPayload})
	}

	return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{outputPayload})
}

func (r *RunPipeline) finish(ctx core.ActionContext) error {
	metadata := RunPipelineExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	if metadata.Execution != nil && (metadata.Execution.Status == PipelineStatusSucceeded || metadata.Execution.Status == PipelineStatusFailed || metadata.Execution.Status == PipelineStatusStopped) {
		return fmt.Errorf("pipeline execution already finished")
	}

	data, ok := ctx.Parameters["data"]
	if !ok {
		data = map[string]any{}
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("data parameter is invalid")
	}

	if metadata.Pipeline == nil {
		return fmt.Errorf("pipeline metadata not found - component may not be properly set up")
	}

	metadata.Extra = dataMap
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	outputPayload := map[string]any{
		"pipeline": map[string]any{
			"name": metadata.Pipeline.Name,
		},
		"manual": true,
		"data":   dataMap,
	}

	if metadata.Execution != nil {
		outputPayload["pipeline"].(map[string]any)["executionId"] = metadata.Execution.ID
		outputPayload["pipeline"].(map[string]any)["status"] = metadata.Execution.Status
	}

	return ctx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{outputPayload})
}

// OnIntegrationMessage receives EventBridge events routed through the AWS
// integration's shared /events endpoint. Unlike triggers, where each message
// starts a new event chain, this component needs to resolve an existing
// execution that is waiting for pipeline completion.
//
// The flow is:
//  1. EventBridge fires a CodePipeline Pipeline Execution State Change event.
//  2. The AWS integration receives it and routes it here via the subscription.
//  3. We match the event to our pipeline, then look up the waiting execution
//     by the pipeline_execution_id KV set during Execute().
//  4. For terminal states (SUCCEEDED/FAILED/CANCELLED), we emit to the
//     appropriate output channel, finishing the execution in near real-time
//     instead of waiting for the next poll cycle.
func (r *RunPipeline) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	event := common.EventBridgeEvent{}
	err := mapstructure.Decode(ctx.Message, &event)
	if err != nil {
		return fmt.Errorf("failed to decode EventBridge event: %w", err)
	}

	pipelineName, ok := event.Detail["pipeline"]
	if !ok {
		return fmt.Errorf("missing pipeline name in event detail")
	}

	name, ok := pipelineName.(string)
	if !ok {
		return fmt.Errorf("invalid pipeline name in event detail")
	}

	metadata := RunPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if metadata.Pipeline == nil {
		return nil
	}

	if name != metadata.Pipeline.Name {
		ctx.Logger.Infof("Skipping event for pipeline %s, expected %s", name, metadata.Pipeline.Name)
		return nil
	}

	state, ok := event.Detail["state"].(string)
	if !ok {
		return nil
	}

	// Only process terminal states; ignore STARTED, RESUMED, etc.
	status, terminal := terminalStatusFromEventBridgeState(state)
	if !terminal {
		return nil
	}

	executionID, ok := event.Detail["execution-id"].(string)
	if !ok || executionID == "" {
		return fmt.Errorf("missing execution-id in EventBridge event detail")
	}

	if ctx.FindExecutionByKV == nil {
		// Fallback: if execution resolution is not available, emit a root event
		// so that the event is at least recorded. The poll will resolve it.
		ctx.Logger.Warnf("FindExecutionByKV not available, falling back to event emission")
		return ctx.Events.Emit(PayloadType, ctx.Message)
	}

	executionCtx, err := ctx.FindExecutionByKV("pipeline_execution_id", executionID)
	if err != nil {
		ctx.Logger.Warnf("Failed to find execution for pipeline_execution_id=%s: %v", executionID, err)
		return nil
	}

	if executionCtx == nil {
		ctx.Logger.Infof("No execution found for pipeline_execution_id=%s, ignoring", executionID)
		return nil
	}

	execMetadata := RunPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &execMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	if execMetadata.Execution == nil {
		return nil
	}

	// Already resolved (e.g., poll beat us to it), nothing to do.
	if execMetadata.Execution.Status == PipelineStatusSucceeded ||
		execMetadata.Execution.Status == PipelineStatusFailed ||
		execMetadata.Execution.Status == PipelineStatusStopped {
		return nil
	}

	if executionCtx.ExecutionState.IsFinished() {
		return nil
	}

	execMetadata.Execution.Status = status
	err = executionCtx.Metadata.Set(execMetadata)
	if err != nil {
		return fmt.Errorf("failed to update execution metadata: %w", err)
	}

	outputPayload := pipelineExecutionOutputPayload(metadata.Pipeline.Name, executionID, status, state, event.Detail)

	if status == PipelineStatusSucceeded {
		return executionCtx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{outputPayload})
	}

	return executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{outputPayload})
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
