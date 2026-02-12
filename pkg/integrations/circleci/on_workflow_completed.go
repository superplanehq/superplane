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

type OnWorkflowCompleted struct{}

type OnWorkflowCompletedMetadata struct {
	Project *Project `json:"project"`
}

type OnWorkflowCompletedConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (p *OnWorkflowCompleted) Name() string {
	return "circleci.onWorkflowCompleted"
}

func (p *OnWorkflowCompleted) Label() string {
	return "On Workflow Completed"
}

func (p *OnWorkflowCompleted) Description() string {
	return "Listen to CircleCI workflow completion events"
}

func (p *OnWorkflowCompleted) Documentation() string {
	return `Triggers when a CircleCI workflow completes.

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

func (p *OnWorkflowCompleted) Icon() string {
	return "workflow"
}

func (p *OnWorkflowCompleted) Color() string {
	return "gray"
}

func (p *OnWorkflowCompleted) Configuration() []configuration.Field {
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

func (p *OnWorkflowCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnWorkflowCompletedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnWorkflowCompletedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	normalizedProjectSlug := strings.TrimSpace(config.ProjectSlug)
	projectChanged := metadata.Project == nil || normalizedProjectSlug != strings.TrimSpace(metadata.Project.Slug)

	if projectChanged {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		project, err := client.GetProject(config.ProjectSlug)
		if err != nil {
			return fmt.Errorf("project not found or inaccessible: %w", err)
		}

		err = ctx.Metadata.Set(OnWorkflowCompletedMetadata{
			Project: &Project{
				ID:   project.ID,
				Slug: project.Slug,
				Name: project.Name,
			},
		})
		if err != nil {
			return fmt.Errorf("error setting metadata: %v", err)
		}
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: config.ProjectSlug,
		Events:      []string{"workflow-completed"},
	})
}

func (p *OnWorkflowCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnWorkflowCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (p *OnWorkflowCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signatureHeader := ctx.Headers.Get("circleci-signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	signature, _ := strings.CutPrefix(signatureHeader, "v1=")

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
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (p *OnWorkflowCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}
