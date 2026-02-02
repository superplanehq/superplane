package semaphore

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

type OnPipelineDone struct{}

type OnPipelineDoneMetadata struct {
	Project *Project `json:"project"`
}

type OnPipelineDoneConfiguration struct {
	Project string `json:"project"`
}

func (p *OnPipelineDone) Name() string {
	return "semaphore.onPipelineDone"
}

func (p *OnPipelineDone) Label() string {
	return "On Pipeline Done"
}

func (p *OnPipelineDone) Description() string {
	return "Listen to Semaphore pipeline done events"
}

func (p *OnPipelineDone) Documentation() string {
	return `The On Pipeline Done trigger starts a workflow execution when a Semaphore pipeline completes.

## Use Cases

- **Pipeline orchestration**: Chain workflows together based on pipeline completion
- **Status monitoring**: Monitor CI/CD pipeline results
- **Notification workflows**: Send notifications when pipelines succeed or fail
- **Post-processing**: Process artifacts or results after pipeline completion

## Configuration

- **Project**: Select the Semaphore project to monitor

## Event Data

Each pipeline done event includes:
- **pipeline**: Pipeline information including ID, state, and result
- **workflow**: Workflow information including ID and URL
- **project**: Project information
- **result**: Pipeline result (passed, failed, stopped, etc.)
- **state**: Pipeline state (done)

## Webhook Setup

This trigger automatically sets up a Semaphore webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPipelineDone) Icon() string {
	return "workflow"
}

func (p *OnPipelineDone) Color() string {
	return "gray"
}

func (p *OnPipelineDone) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
		},
	}
}

func (p *OnPipelineDone) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineDoneMetadata
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

	config := OnPipelineDoneConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && (config.Project == metadata.Project.ID || config.Project == metadata.Project.Name) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.Project, err)
	}

	err = ctx.Metadata.Set(OnPipelineDoneMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Project: project.Metadata.ProjectName,
	})
}

func (p *OnPipelineDone) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipelineDone) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPipelineDone) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Semaphore-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
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

	err = ctx.Events.Emit("semaphore.pipeline.done", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPipelineDone) Cleanup(ctx core.TriggerContext) error {
	return nil
}
