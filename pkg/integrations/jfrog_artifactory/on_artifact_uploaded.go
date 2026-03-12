package jfrogartifactory

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// OnArtifactUploaded is a trigger that fires when an artifact is deployed to JFrog Artifactory.
type OnArtifactUploaded struct{}

// OnArtifactUploadedConfiguration holds the trigger configuration.
type OnArtifactUploadedConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

// ArtifactDeployedData holds the artifact data from the JFrog webhook payload.
type ArtifactDeployedData struct {
	RepoKey string `json:"repo_key"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

// OnArtifactUploadedPayload is the incoming JFrog webhook payload structure.
type OnArtifactUploadedPayload struct {
	Domain    string               `json:"domain"`
	EventType string               `json:"event_type"`
	Data      ArtifactDeployedData `json:"data"`
}

func (t *OnArtifactUploaded) Name() string {
	return "jfrogArtifactory.onArtifactUploaded"
}

func (t *OnArtifactUploaded) Label() string {
	return "On Artifact Uploaded"
}

func (t *OnArtifactUploaded) Description() string {
	return "Triggers when an artifact is deployed to JFrog Artifactory"
}

func (t *OnArtifactUploaded) Documentation() string {
	return `The On Artifact Uploaded trigger starts a workflow execution when an artifact is deployed to JFrog Artifactory.

## Configuration

- **Repository** (optional): Filter events to a specific repository. Leave empty to trigger for all repositories.

## Outputs

- **Default channel**: Emits artifact deploy data including repo, path, name, size, and sha256.`
}

func (t *OnArtifactUploaded) Icon() string {
	return "jfrogArtifactory"
}

func (t *OnArtifactUploaded) Color() string {
	return "green"
}

func (t *OnArtifactUploaded) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "repository",
				},
			},
		},
	}
}

func (t *OnArtifactUploaded) Setup(ctx core.TriggerContext) error {
	var config OnArtifactUploadedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Repository: config.Repository,
	})
}

func (t *OnArtifactUploaded) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-JFrog-Event-Auth")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-JFrog-Event-Auth header")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid webhook signature: %v", err)
	}

	var payload OnArtifactUploadedPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if payload.EventType != "deployed" {
		return http.StatusOK, nil
	}

	var config OnArtifactUploadedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository != "" && payload.Data.RepoKey != config.Repository {
		return http.StatusOK, nil
	}

	flatPayload := map[string]any{
		"repo":   payload.Data.RepoKey,
		"path":   payload.Data.Path,
		"name":   payload.Data.Name,
		"size":   payload.Data.Size,
		"sha256": payload.Data.SHA256,
	}

	if err := ctx.Events.Emit("jfrogArtifactory.artifactUploaded", flatPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnArtifactUploaded) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnArtifactUploaded) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnArtifactUploaded) Cleanup(ctx core.TriggerContext) error {
	return nil
}
