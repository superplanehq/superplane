package dockerhub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnImagePushed struct{}

type OnImagePushedConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	TagFilter  string `json:"tagFilter" mapstructure:"tagFilter"`
}

type OnImagePushedMetadata struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

func (t *OnImagePushed) Name() string {
	return "dockerhub.onImagePushed"
}

func (t *OnImagePushed) Label() string {
	return "On Image Pushed"
}

func (t *OnImagePushed) Description() string {
	return "Trigger when a new image is pushed to Docker Hub"
}

func (t *OnImagePushed) Documentation() string {
	return `The On Image Pushed trigger starts a workflow when a new image is pushed to a Docker Hub repository.

## Use Cases

- **Deploy to staging or production**: Automatically deploy when a new image is pushed
- **Notify team**: Send Slack notifications when base images are updated
- **Trigger scans**: Run vulnerability scans when images are pushed
- **Promote images**: Trigger promotion workflows when images are pushed to a repository

## Configuration

- **Repository**: Docker Hub repository to watch (e.g., ` + "`myorg/myapp`" + `)
- **Tag Filter**: Only trigger for tags matching a pattern (optional, e.g., ` + "`v*`" + `, ` + "`main`" + `)

## Event Data

Each push event includes:
- **repository**: Repository information (name, namespace, owner, etc.)
- **push_data**: Push details including tag, pusher, and timestamp
- **callback_url**: Docker Hub callback URL

## Webhook Setup

1. Go to your Docker Hub repository settings
2. Navigate to Webhooks
3. Add a new webhook with the URL provided by SuperPlane
4. The trigger will receive events when images are pushed
`
}

func (t *OnImagePushed) Icon() string {
	return "docker"
}

func (t *OnImagePushed) Color() string {
	return "gray"
}

func (t *OnImagePushed) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Docker Hub repository to watch (e.g., myorg/myapp)",
			Placeholder: "myorg/myapp",
		},
		{
			Name:        "tagFilter",
			Label:       "Tag Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only trigger for tags matching this pattern (e.g., v*, main)",
			Placeholder: "v*",
		},
	}
}

func (t *OnImagePushed) Setup(ctx core.TriggerContext) error {
	var metadata OnImagePushedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnImagePushedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	// If metadata matches current config, trigger is already setup
	if metadata.Repository == config.Repository {
		return nil
	}

	// Request webhook setup first - only persist metadata after success
	err = ctx.Integration.RequestWebhook(WebhookConfiguration{
		Repository: config.Repository,
	})
	if err != nil {
		return err
	}

	// Store metadata only after webhook request succeeds
	metadata.Repository = config.Repository
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (t *OnImagePushed) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnImagePushed) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnImagePushed) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnImagePushedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Parse the webhook payload
	var payload WebhookPayload
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Verify this is for the configured repository
	if payload.Repository.RepoName != config.Repository {
		return http.StatusOK, nil
	}

	// Apply tag filter if configured
	if config.TagFilter != "" {
		if !matchesPattern(payload.PushData.Tag, config.TagFilter) {
			return http.StatusOK, nil
		}
	}

	// Convert payload to map for emission
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Emit the event
	err = ctx.Events.Emit("dockerhub.imagePushed", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnImagePushed) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// matchesPattern checks if a string matches a simple wildcard pattern
// Supports * for wildcard matching
func matchesPattern(s, pattern string) bool {
	if pattern == "" {
		return true
	}

	// Handle contains matching (e.g., "*beta*") - check this FIRST
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 2 {
		contains := pattern[1 : len(pattern)-1]
		return strings.Contains(s, contains)
	}

	// Handle simple prefix matching (e.g., "v*")
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}

	// Handle simple suffix matching (e.g., "*-latest")
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(s, suffix)
	}

	// Exact match
	return s == pattern
}
