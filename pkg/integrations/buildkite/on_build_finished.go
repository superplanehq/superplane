package buildkite

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBuildFinished struct{}

type OnBuildFinishedMetadata struct {
	Pipeline          string  `json:"pipeline"`
	Branch            string  `json:"branch,omitempty"`
	AppSubscriptionID *string `json:"appSubscriptionID,omitempty"`
}

type OnBuildFinishedConfiguration struct {
	Pipeline string `json:"pipeline"`
	Branch   string `json:"branch,omitempty"`
}

type BuildkiteSubscriptionConfiguration struct {
	Organization string `json:"organization"`
	Pipeline     string `json:"pipeline"`
	Branch       string `json:"branch,omitempty"`
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

This trigger automatically handles Buildkite webhook events through the integration-level webhook. When you configure a single webhook for your Buildkite integration, SuperPlane will automatically route build.finished events to all matching triggers based on your configuration.

## Event Processing

SuperPlane automatically:
1. Receives webhook events at integration webhook URL
2. Authenticates requests using your webhook token
3. Filters events by organization, pipeline, and branch
4. Routes matching events to appropriate trigger instances
5. Emits buildkite.build.finished events to start workflow executions

## Manual Webhook Configuration (if needed)

For manual setup in Buildkite:
1. In Buildkite, go to Settings → Notification Services → Add → Webhook
2. Webhook URL: use your SuperPlane integration webhook URL
3. Events: select "build.finished"
4. Pipelines: select the pipelines you want to monitor`
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
	// If subscription ID is already set, nothing to do.
	var metadata OnBuildFinishedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Validate configuration
	var config OnBuildFinishedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate that we have the required configuration
	if config.Pipeline == "" {
		return fmt.Errorf("pipeline is required for trigger configuration")
	}

	subscriptionID, err := t.subscribe(ctx, metadata, config)
	if err != nil {
		return fmt.Errorf("failed to subscribe to buildkite events: %w", err)
	}

	return ctx.Metadata.Set(OnBuildFinishedMetadata{
		Pipeline:          config.Pipeline,
		Branch:            config.Branch,
		AppSubscriptionID: subscriptionID,
	})
}

func (t *OnBuildFinished) subscribe(ctx core.TriggerContext, metadata OnBuildFinishedMetadata, config OnBuildFinishedConfiguration) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		return metadata.AppSubscriptionID, nil
	}

	orgConfig, err := ctx.Integration.GetConfig("organization")
	if err != nil {
		return nil, fmt.Errorf("failed to get organization from integration config: %w", err)
	}
	orgSlug, err := extractOrgSlug(string(orgConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to extract organization slug: %w", err)
	}

	subscriptionID, err := ctx.Integration.Subscribe(BuildkiteSubscriptionConfiguration{
		Organization: orgSlug,
		Pipeline:     config.Pipeline,
		Branch:       config.Branch,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to buildkite events: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnBuildFinished) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBuildFinished) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBuildFinished) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnBuildFinished) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnBuildFinished) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	// Extract webhook configuration from context
	var config OnBuildFinishedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Extract message payload (should be build.finished event)
	message := ctx.Message
	if message == nil {
		return fmt.Errorf("received empty message")
	}

	// Verify this is a build.finished event
	messageMap, ok := message.(map[string]any)
	if !ok {
		return fmt.Errorf("message is not a map")
	}

	eventType, ok := messageMap["event"].(string)
	if !ok || eventType != "build.finished" {
		return nil // Silently ignore non-build.finished events
	}

	// Emit event to trigger workflow execution
	err := ctx.Events.Emit("buildkite.build.finished", message)
	if err != nil {
		return fmt.Errorf("error emitting event: %v", err)
	}

	return nil
}
