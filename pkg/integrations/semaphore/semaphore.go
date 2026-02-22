package semaphore

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("semaphore", &Semaphore{}, &SemaphoreWebhookHandler{})
}

type Semaphore struct{}

type Configuration struct {
	OrganizationURL string `json:"organizationUrl"`
	APIToken        string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (s *Semaphore) Name() string {
	return "semaphore"
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Icon() string {
	return "workflow"
}

func (s *Semaphore) Description() string {
	return "Run and react to your Semaphore workflows"
}

func (s *Semaphore) Instructions() string {
	return ""
}

func (s *Semaphore) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organizationUrl",
			Label:       "Organization URL",
			Type:        configuration.FieldTypeString,
			Description: "Semaphore organization URL",
			Placeholder: "e.g. https://superplane.semaphoreci.com",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Semaphore API token",
			Required:    true,
		},
	}
}

func (s *Semaphore) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Semaphore) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("Failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	//
	// Semaphore doesn't have a whoami endpoint, so
	// we list projects just to verify that the connection is working.
	//
	_, err = client.listProjects()
	if err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (s *Semaphore) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *Semaphore) Actions() []core.Action {
	return []core.Action{}
}

func (s *Semaphore) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Semaphore) Components() []core.Component {
	return []core.Component{
		&RunWorkflow{},
		&GetPipeline{},
	}
}

func (s *Semaphore) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineDone{},
	}
}
