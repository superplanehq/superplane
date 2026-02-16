package jenkins

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBuildFinished struct{}

type OnBuildFinishedConfiguration struct {
	Job string `json:"job"`
}

type OnBuildFinishedMetadata struct {
	Job *JobInfo `json:"job" mapstructure:"job"`
}

func (t *OnBuildFinished) Name() string {
	return "jenkins.onBuildFinished"
}

func (t *OnBuildFinished) Label() string {
	return "On Build Finished"
}

func (t *OnBuildFinished) Description() string {
	return "Listen to Jenkins build completion events"
}

func (t *OnBuildFinished) Documentation() string {
	return `Triggers when a Jenkins build completes.

## Use Cases

- **Deployment pipelines**: Start deployments when CI builds succeed
- **Notifications**: Send alerts when builds fail
- **Workflow chaining**: Chain multiple Jenkins jobs through SuperPlane

## Configuration

- **Job**: Select the Jenkins job to monitor

## Event Data

Each build completion event includes:
- **job**: Job name and URL
- **build**: Build number, URL, and result (SUCCESS, FAILURE, UNSTABLE, ABORTED)

## Webhook Setup

This trigger requires the Jenkins Notification Plugin. Configure it in your Jenkins job
to POST build events to the webhook URL displayed after saving the canvas.`
}

func (t *OnBuildFinished) Icon() string {
	return "jenkins"
}

func (t *OnBuildFinished) Color() string {
	return "gray"
}

func (t *OnBuildFinished) Configuration() []configuration.Field {
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
	}
}

func (t *OnBuildFinished) Setup(ctx core.TriggerContext) error {
	config := OnBuildFinishedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Job == "" {
		return fmt.Errorf("job is required")
	}

	var metadata OnBuildFinishedMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If already set up for the same job, skip re-setup.
	if metadata.Job != nil && metadata.Job.Name == config.Job {
		return ctx.Integration.RequestWebhook(WebhookConfiguration{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	job, err := client.GetJob(config.Job)
	if err != nil {
		return fmt.Errorf("error finding job %s: %v", config.Job, err)
	}

	if err := ctx.Metadata.Set(OnBuildFinishedMetadata{
		Job: &JobInfo{
			Name: job.FullName,
			URL:  job.URL,
		},
	}); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{})
}

func (t *OnBuildFinished) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
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

	config := OnBuildFinishedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Job != "" && payload.Name != config.Job {
		return http.StatusOK, nil
	}

	eventPayload := map[string]any{
		"job": map[string]any{
			"name": payload.Name,
			"url":  payload.URL,
		},
		"build": map[string]any{
			"number": payload.Build.Number,
			"url":    payload.Build.FullURL,
			"result": payload.Build.Status,
		},
	}

	if err := ctx.Events.Emit(PayloadType, eventPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnBuildFinished) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBuildFinished) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnBuildFinished) Cleanup(ctx core.TriggerContext) error {
	return nil
}
