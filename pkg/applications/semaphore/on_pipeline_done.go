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
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (p *OnPipelineDone) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineDoneMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
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

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.Project, err)
	}

	ctx.MetadataContext.Set(OnPipelineDoneMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Project: project.Metadata.ProjectName,
	})
}

func (p *OnPipelineDone) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipelineDone) HandleAction(ctx core.TriggerActionContext) error {
	return nil
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

	secret, err := ctx.WebhookContext.GetSecret()
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

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
