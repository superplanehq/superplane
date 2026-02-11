package circleci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnPipelineCompleted struct{}

type OnPipelineCompletedMetadata struct {
	Project *Project `json:"project"`
}

type OnPipelineCompletedConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (p *OnPipelineCompleted) Name() string {
	return "circleci.onPipelineCompleted"
}

func (p *OnPipelineCompleted) Label() string {
	return "On Pipeline Completed"
}

func (p *OnPipelineCompleted) Description() string {
	return "Listen to CircleCI workflow completion events within a pipeline"
}

func (p *OnPipelineCompleted) Documentation() string {
	return `Triggers when all CircleCI workflow completes within a pipeline.

## Use Cases

- **Workflow chaining**: Start SuperPlane workflows when CircleCI workflows complete
- **Status monitoring**: Monitor CI/CD workflow results
- **Notifications**: Send alerts when workflows succeed or fail
- **Post-processing**: Process artifacts after workflow completion

## Configuration

- **Project Slug**: The CircleCI project slug (e.g., gh/username/repo)

## Event Data

Each workflow completion event includes:
- **workflow**: Workflow information including ID, name, status, and URL
- **pipeline**: Parent pipeline information including ID, number, and trigger details
- **project**: Project information
- **organization**: Organization information

## Webhook Setup

This trigger automatically sets up a CircleCI webhook when configured. The webhook is managed by SuperPlane and cleaned up when the trigger is removed.`
}

func (p *OnPipelineCompleted) Icon() string {
	return "workflow"
}

func (p *OnPipelineCompleted) Color() string {
	return "gray"
}

func (p *OnPipelineCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g. gh/org/repo). Find in CircleCI project settings.",
		},
	}
}

func (p *OnPipelineCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineCompletedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnPipelineCompletedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	if metadata.Project != nil && config.ProjectSlug == metadata.Project.Slug {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	project, err := client.GetProject(config.ProjectSlug)
	if err != nil {
		return fmt.Errorf("project not found or inaccessible: %w", err)
	}

	err = ctx.Metadata.Set(OnPipelineCompletedMetadata{
		Project: &Project{
			ID:   project.ID,
			Slug: project.Slug,
			Name: project.Name,
		},
	})
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: config.ProjectSlug,
		Events:      []string{"workflow-completed"},
	})
}

func (p *OnPipelineCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipelineCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPipelineCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signatureHeader := ctx.Headers.Get("circleci-signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	signature := signatureHeader
	if strings.HasPrefix(signatureHeader, "v1=") {
		signature = strings.TrimPrefix(signatureHeader, "v1=")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType, ok := data["type"].(string)
	if !ok || eventType != "workflow-completed" {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("circleci.workflow.completed", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPipelineCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}
