package jenkins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	PayloadType         = "jenkins.build.finished"
	PassedOutputChannel = "passed"
	FailedOutputChannel = "failed"
	BuildResultSuccess  = "SUCCESS"
	PollInterval        = 1 * time.Minute
	QueuePollInterval   = 15 * time.Second
	buildJobKey         = "buildJob"
)

type TriggerBuild struct{}

type TriggerBuildSpec struct {
	Job        string      `json:"job"`
	Parameters []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TriggerBuildNodeMetadata struct {
	Job *JobInfo `json:"job" mapstructure:"job"`
}

type TriggerBuildExecutionMetadata struct {
	Job         *JobInfo   `json:"job" mapstructure:"job"`
	QueueItemID int64      `json:"queueItemId" mapstructure:"queueItemId"`
	Build       *BuildInfo `json:"build,omitempty" mapstructure:"build,omitempty"`
}

type JobInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type BuildInfo struct {
	Number   int64  `json:"number"`
	URL      string `json:"url"`
	Result   string `json:"result"`
	Building bool   `json:"building"`
}

// webhookPayload represents the Jenkins Notification Plugin payload.
type webhookPayload struct {
	Name  string        `json:"name"`
	URL   string        `json:"url"`
	Build *webhookBuild `json:"build"`
}

type webhookBuild struct {
	FullURL string `json:"full_url"`
	Number  int64  `json:"number"`
	Phase   string `json:"phase"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

func (t *TriggerBuild) Name() string {
	return "jenkins.triggerBuild"
}

func (t *TriggerBuild) Label() string {
	return "Trigger Build"
}

func (t *TriggerBuild) Description() string {
	return "Trigger a Jenkins build and wait for completion"
}

func (t *TriggerBuild) Documentation() string {
	return `The Trigger Build component triggers a Jenkins job and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds and deployments from SuperPlane workflows
- **Pipeline automation**: Run Jenkins jobs as part of workflow automation
- **Multi-stage deployments**: Coordinate complex deployment pipelines

## How It Works

1. Triggers the specified Jenkins job with optional parameters
2. Waits for the build to complete (via webhook from Jenkins Notification Plugin, with polling as fallback)
3. Routes execution based on build result:
   - **Passed channel**: Build completed with SUCCESS
   - **Failed channel**: Build failed, was unstable, or was aborted

## Configuration

- **Job**: Select the Jenkins job to trigger
- **Parameters**: Optional build parameters as key-value pairs (supports expressions)

## Output Channels

- **Passed**: Emitted when build completes with SUCCESS
- **Failed**: Emitted when build fails, is unstable, or is aborted`
}

func (t *TriggerBuild) Icon() string {
	return "jenkins"
}

func (t *TriggerBuild) Color() string {
	return "gray"
}

func (t *TriggerBuild) OutputChannels(configuration any) []core.OutputChannel {
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

func (t *TriggerBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "job",
			Label:    "Job",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "job",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:  "parameters",
			Label: "Parameters",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Parameter",
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
	}
}

func (t *TriggerBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *TriggerBuild) Setup(ctx core.SetupContext) error {
	spec := TriggerBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Job == "" {
		return fmt.Errorf("job is required")
	}

	metadata := TriggerBuildNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If already set up for the same job, skip re-setup.
	if metadata.Job != nil && metadata.Job.Name == spec.Job {
		return ctx.Integration.RequestWebhook(WebhookConfiguration{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	job, err := client.GetJob(spec.Job)
	if err != nil {
		return fmt.Errorf("error finding job %s: %v", spec.Job, err)
	}

	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{}); err != nil {
		return err
	}

	return ctx.Metadata.Set(TriggerBuildNodeMetadata{
		Job: &JobInfo{
			Name: job.FullName,
			URL:  job.URL,
		},
	})
}

func (t *TriggerBuild) Execute(ctx core.ExecutionContext) error {
	spec := TriggerBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	nodeMetadata := TriggerBuildNodeMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	params := make(map[string]string)
	for _, p := range spec.Parameters {
		params[p.Name] = p.Value
	}

	queueItemID, err := client.TriggerBuild(spec.Job, params)
	if err != nil {
		return fmt.Errorf("error triggering build: %v", err)
	}

	ctx.Logger.Infof("Build triggered - job=%s, queueItem=%d", spec.Job, queueItemID)

	if err := ctx.Metadata.Set(TriggerBuildExecutionMetadata{
		Job:         nodeMetadata.Job,
		QueueItemID: queueItemID,
	}); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV("queueItem", fmt.Sprintf("%d", queueItemID)); err != nil {
		return err
	}

	// NOTE: The job name is not unique per execution. If multiple executions
	// trigger builds of the same job concurrently, the webhook may match the
	// wrong execution. The polling fallback handles this correctly since each
	// execution tracks its own queueItemID. The Jenkins Notification Plugin
	// payload does not include queue item IDs, so webhook matching is limited
	// to job name for now.
	if err := ctx.ExecutionState.SetKV(buildJobKey, spec.Job); err != nil {
		return err
	}

	// Wait for webhook from Jenkins Notification Plugin; poll as fallback.
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, QueuePollInterval)
}

func (t *TriggerBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TriggerBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	payload := webhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	if payload.Build == nil {
		return http.StatusOK, nil
	}

	if payload.Build.Phase != "COMPLETED" && payload.Build.Phase != "FINALIZED" {
		return http.StatusOK, nil
	}

	if payload.Name == "" || ctx.FindExecutionByKV == nil {
		return http.StatusOK, nil
	}

	executionCtx, err := ctx.FindExecutionByKV(buildJobKey, payload.Name)
	if err != nil {
		return http.StatusOK, nil
	}
	if executionCtx == nil {
		return http.StatusOK, nil
	}

	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}

	metadata := TriggerBuildExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %w", err)
	}

	if metadata.Job == nil {
		return http.StatusInternalServerError, fmt.Errorf("metadata.Job is nil for execution")
	}

	metadata.Build = &BuildInfo{
		Number:   payload.Build.Number,
		URL:      payload.Build.FullURL,
		Result:   payload.Build.Status,
		Building: false,
	}

	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, err
	}

	emitPayload := map[string]any{
		"job": map[string]any{
			"name": payload.Name,
			"url":  metadata.Job.URL,
		},
		"build": map[string]any{
			"number": payload.Build.Number,
			"url":    payload.Build.FullURL,
			"result": payload.Build.Status,
		},
	}

	if payload.Build.Status == BuildResultSuccess {
		if err := executionCtx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{emitPayload}); err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	}

	if err := executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{emitPayload}); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *TriggerBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (t *TriggerBuild) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return t.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *TriggerBuild) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec := TriggerBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	metadata := TriggerBuildExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Job == nil {
		return fmt.Errorf("metadata.Job is nil for execution")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Phase 1: build is still in queue, waiting for a build number.
	if metadata.Build == nil || metadata.Build.Number == 0 {
		queueItem, err := client.GetQueueItem(metadata.QueueItemID)
		if err != nil {
			return fmt.Errorf("error getting queue item: %v", err)
		}

		if queueItem.Executable == nil || queueItem.Executable.Number == 0 {
			return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, QueuePollInterval)
		}

		metadata.Build = &BuildInfo{
			Number:   queueItem.Executable.Number,
			URL:      queueItem.Executable.URL,
			Building: true,
		}

		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
	}

	// Phase 2: build is running, poll for completion.
	build, err := client.GetBuild(spec.Job, metadata.Build.Number)
	if err != nil {
		return fmt.Errorf("error getting build: %v", err)
	}

	if build.Building {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	// Build finished -- update metadata and emit result.
	metadata.Build.Result = build.Result
	metadata.Build.Building = false
	metadata.Build.URL = build.URL
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	payload := map[string]any{
		"job": map[string]any{
			"name": spec.Job,
			"url":  metadata.Job.URL,
		},
		"build": map[string]any{
			"number":   build.Number,
			"url":      build.URL,
			"result":   build.Result,
			"duration": build.Duration,
		},
	}

	if build.Result == BuildResultSuccess {
		return ctx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
}

func (t *TriggerBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}
