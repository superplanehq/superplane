package buildkite

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBuildFinished struct{}

type OnBuildFinishedMetadata struct {
	Pipeline     string `json:"pipeline"`
	Branch       string `json:"branch,omitempty"`
	WebhookURL   string `json:"webhookUrl"`
	WebhookToken string `json:"webhookToken"`
	OrgSlug      string `json:"orgSlug"`
}

type OnBuildFinishedConfiguration struct {
	Pipeline string `json:"pipeline"`
	Branch   string `json:"branch,omitempty"`
}

func (t *OnBuildFinished) Name() string {
	return "buildkite.onBuildFinished"
}

func (t *OnBuildFinished) Label() string {
	return "On Build Finished"
}

func (t *OnBuildFinished) Description() string {
	return "Listen to Buildkite build completion events"
}

func (t *OnBuildFinished) Documentation() string {
	return `The On Build Finished trigger starts a workflow execution when a Buildkite build completes.

## Use Cases

- **CI/CD pipeline chaining**: Trigger downstream workflows when builds complete
- **Build monitoring**: Monitor build results and trigger alerts or notifications
- **Deployment orchestration**: Start deployment workflows only after successful builds
- **Build result processing**: Process build artifacts or results based on build outcome

## Configuration

- **Pipeline**: Select the Buildkite pipeline to monitor
- **Branch** (optional): Filter to specific branch (exact match)

## Event Data

Each build finished event includes:
- **build**: Build information including ID, state, result, commit, branch
- **pipeline**: Pipeline information including ID and name
- **organization**: Organization information
- **sender**: User who triggered the build

## Webhook Setup

This trigger requires setting up a webhook in Buildkite to receive build events:

1. When you configure this trigger, SuperPlane generates a unique webhook URL and token
2. A browser action will guide you to the Buildkite webhook settings page
3. In Buildkite, create a new webhook with:
   - **Webhook URL**: The URL provided by SuperPlane
   - **Webhook Token**: The token provided by SuperPlane
   - **Events**: Select "build.finished"
   - **Pipelines**: Select the specific pipeline(s) you want to monitor

## Event Processing

SuperPlane automatically:
1. Receives webhook events at the trigger-specific webhook URL
2. Authenticates requests using the webhook token
3. Filters events by pipeline and branch (if configured)
4. Emits buildkite.build.finished events to start workflow executions`
}

func (t *OnBuildFinished) Icon() string {
	return "workflow"
}

func (t *OnBuildFinished) Color() string {
	return "gray"
}

func (t *OnBuildFinished) ExampleData() map[string]any {
	return map[string]any{
		"event": "build.finished",
		"build": map[string]any{
			"id":      "12345678-1234-1234-1234-123456789012",
			"number":  123,
			"state":   "passed",
			"web_url": "https://buildkite.com/example-org/example-pipeline/builds/123",
			"commit":  "a1b2c3d4e5f6789012345678901234567890abcd",
			"branch":  "main",
			"message": "Fix: Update dependencies",
			"blocked": false,
		},
		"pipeline": map[string]any{
			"id":   "example-pipeline",
			"name": "Example Pipeline",
		},
		"organization": map[string]any{
			"id":   "example-org",
			"name": "Example Organization",
		},
		"sender": map[string]any{
			"id":    "user-123",
			"name":  "John Doe",
			"email": "john@example.com",
		},
	}
}

func (t *OnBuildFinished) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "pipeline",
			Label:    "Pipeline",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pipeline",
				},
			},
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Optional: Filter to specific branch (exact match)",
			Placeholder: "e.g. main, develop",
		},
	}
}

func (t *OnBuildFinished) Setup(ctx core.TriggerContext) error {
	metadata := OnBuildFinishedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnBuildFinishedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}

	// If webhook is already set up for this pipeline, nothing to do
	if metadata.Pipeline == config.Pipeline &&
		metadata.Branch == config.Branch &&
		metadata.WebhookURL != "" &&
		metadata.WebhookToken != "" {
		return nil
	}

	// Get orgSlug from integration config
	orgConfig, err := ctx.Integration.GetConfig("organization")
	if err != nil {
		return fmt.Errorf("failed to get organization from integration config: %w", err)
	}
	orgSlug, err := extractOrgSlug(string(orgConfig))
	if err != nil {
		return fmt.Errorf("failed to extract organization slug: %w", err)
	}

	var webhookSecret []byte
	webhookURL := metadata.WebhookURL

	if webhookURL == "" {
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}
		webhookSecret, err = ctx.Webhook.GetSecret()
		if err != nil {
			return fmt.Errorf("failed to get webhook secret: %w", err)
		}
	} else {
		webhookSecret = []byte(metadata.WebhookToken)
	}

	return ctx.Metadata.Set(OnBuildFinishedMetadata{
		Pipeline:     config.Pipeline,
		Branch:       config.Branch,
		WebhookURL:   webhookURL,
		WebhookToken: string(webhookSecret),
		OrgSlug:      orgSlug,
	})
}

func (t *OnBuildFinished) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBuildFinished) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBuildFinished) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnBuildFinishedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnBuildFinishedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Verify webhook signature/token
	if err := VerifyWebhook(ctx.Headers, ctx.Body, []byte(metadata.WebhookToken)); err != nil {
		ctx.Logger.WithError(err).Warn("webhook verification failed")
		return http.StatusForbidden, fmt.Errorf("webhook verification failed: %w", err)
	}

	// Parse the payload
	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	// Verify this is a build.finished event
	eventType := ctx.Headers.Get("X-Buildkite-Event")
	if eventType == "" {
		if event, ok := payload["event"].(string); ok {
			eventType = event
		}
	}

	if eventType != "build.finished" {
		return http.StatusOK, nil // Silently ignore non-build.finished events
	}

	// Filter by pipeline if configured
	if config.Pipeline != "" && config.Pipeline != "*" {
		if pipeline, ok := payload["pipeline"].(map[string]any); ok {
			if pipelineSlug, ok := pipeline["slug"].(string); ok {
				if config.Pipeline != pipelineSlug {
					ctx.Logger.Infof("Ignoring event for pipeline %s", pipelineSlug)
					return http.StatusOK, nil
				}
			}
		}
	}

	// Filter by branch if configured
	if config.Branch != "" {
		if build, ok := payload["build"].(map[string]any); ok {
			if branch, ok := build["branch"].(string); ok {
				if config.Branch != branch {
					ctx.Logger.Infof("Ignoring event for branch %s", branch)
					return http.StatusOK, nil
				}
			}
		}
	}

	// Emit event to trigger workflow execution
	if err := ctx.Events.Emit("buildkite.build.finished", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnBuildFinished) Cleanup(ctx core.TriggerContext) error {
	return nil
}
