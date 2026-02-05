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
	Project *ProjectInfo `json:"project"`
}

type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

type OnWorkflowCompletedConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

func (t *OnWorkflowCompleted) Name() string {
	return "circleci.onWorkflowCompleted"
}

func (t *OnWorkflowCompleted) Label() string {
	return "On Workflow Completed"
}

func (t *OnWorkflowCompleted) Description() string {
	return "Listen to CircleCI workflow completion events"
}

func (t *OnWorkflowCompleted) Documentation() string {
	return `The On Workflow Completed trigger starts a workflow execution when a CircleCI workflow completes.

## Use Cases

- **Pipeline orchestration**: Chain workflows together based on CircleCI workflow completion
- **Status monitoring**: Monitor CI/CD workflow results
- **Notification workflows**: Send notifications when workflows succeed or fail
- **Post-processing**: Process artifacts or results after workflow completion

## Configuration

- **Project Slug**: The CircleCI project slug (e.g., gh/org/repo or circleci/org-id/project-id)

## Event Data

Each workflow completed event includes:
- **workflow**: Workflow information including ID, name, status, and URL
- **pipeline**: Pipeline information including ID and number
- **project**: Project information including ID, name, and slug
- **organization**: Organization information

## Webhook Setup

This trigger automatically sets up a CircleCI webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnWorkflowCompleted) Icon() string {
	return "circleci"
}

func (t *OnWorkflowCompleted) Color() string {
	return "gray"
}

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return map[string]any{
		"type": "workflow-completed",
		"workflow": map[string]any{
			"id":         "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
			"name":       "build-test-deploy",
			"status":     "success",
			"created_at": "2021-09-01T22:49:03.616Z",
			"stopped_at": "2021-09-01T22:49:34.170Z",
			"url":        "https://app.circleci.com/pipelines/github/circleci/webhook-service/130/workflows/fda08377-fe7e-46b1-8992-3a7aaecac9c3",
		},
		"pipeline": map[string]any{
			"id":         "1285fe1d-d3a6-44fc-8886-8979558254c4",
			"number":     130,
			"created_at": "2021-09-01T22:49:03.544Z",
		},
		"project": map[string]any{
			"id":   "84996744-a854-4f5e-aea3-04e2851dc1d2",
			"name": "webhook-service",
			"slug": "github/circleci/webhook-service",
		},
		"organization": map[string]any{
			"id":   "f22b6566-597d-46d5-ba74-99ef5bb3d85c",
			"name": "circleci",
		},
	}
}

func (t *OnWorkflowCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g., gh/org/repo)",
			Placeholder: "e.g. gh/myorg/myrepo",
		},
	}
}

func (t *OnWorkflowCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnWorkflowCompletedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// If metadata is set, it means the trigger was already setup
	//
	if metadata.Project != nil {
		return nil
	}

	config := OnWorkflowCompletedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && config.ProjectSlug == metadata.Project.Slug {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.ProjectSlug)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.ProjectSlug, err)
	}

	err = ctx.Metadata.Set(OnWorkflowCompletedMetadata{
		Project: &ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
			Slug: project.Slug,
			URL:  fmt.Sprintf("https://app.circleci.com/pipelines/%s", project.Slug),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: project.Slug,
	})
}

func (t *OnWorkflowCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnWorkflowCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnWorkflowCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// CircleCI uses circleci-signature header for webhook verification
	signature := ctx.Headers.Get("circleci-signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature header")
	}

	// Parse the signature - format is "v1=<hash>"
	signatureParts := strings.Split(signature, "=")
	if len(signatureParts) != 2 || signatureParts[0] != "v1" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signatureParts[1]); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Check if this is a workflow-completed event
	eventType, ok := data["type"].(string)
	if !ok || eventType != "workflow-completed" {
		// Silently ignore other event types
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("circleci.workflow.completed", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnWorkflowCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}
