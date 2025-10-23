package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/triggers"
)

const MaxEventSize = 64 * 1024

type Semaphore struct{}

type Metadata struct {
	Project *Project `json:"project"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Configuration struct {
	Integration string `json:"integration"`
	Project     string `json:"project"`
}

func (s *Semaphore) Name() string {
	return "semaphore"
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Description() string {
	return "Start a new execution chain when something happens in your Semaphore project"
}

func (s *Semaphore) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "Semaphore integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "semaphore",
				},
			},
		},
		{
			Name:     "project",
			Label:    "Project",
			Type:     components.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []components.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
			TypeOptions: &components.TypeOptions{
				Resource: &components.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
	}
}

func (s *Semaphore) Setup(ctx triggers.TriggerContext) error {
	var metadata Metadata
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

	config := Configuration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Integration == "" {
		return fmt.Errorf("integration is required")
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	integration, err := ctx.IntegrationContext.GetIntegration(config.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	resource, err := integration.Get("project", config.Project)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", config.Project, err)
	}

	integrationID, err := uuid.Parse(config.Integration)
	if err != nil {
		return fmt.Errorf("integration ID is invalid: %w", err)
	}

	err = ctx.WebhookContext.Setup(&triggers.WebhookSetupOptions{
		IntegrationID: &integrationID,
		Resource:      resource,
		Configuration: config,
	})

	if err != nil {
		return err
	}

	ctx.MetadataContext.Set(Metadata{
		Project: &Project{
			ID:   resource.Id(),
			Name: resource.Name(),
			URL:  "",
		},
	})

	return nil
}

func (s *Semaphore) Actions() []components.Action {
	return []components.Action{}
}

func (s *Semaphore) HandleAction(ctx triggers.TriggerActionContext) error {
	return nil
}

func (s *Semaphore) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
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
