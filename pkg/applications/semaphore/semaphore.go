package semaphore

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

func init() {
	registry.RegisterApplication("semaphore", &Semaphore{})
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

func (s *Semaphore) Sync(ctx applications.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppContext.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("Failed to decode metadata: %v", err)
	}

	//
	// TODO: Decrypt the API token to validate it can be decrypted
	// TODO: list projects to check if credentials are correct.
	// TODO: save projects in metadata? What if they change?
	//

	ctx.AppContext.SetState("ready")
	return nil
}

func (s *Semaphore) HandleRequest(ctx applications.HttpRequestContext) {
	// TODO: no op?
}

func (s *Semaphore) Components() []components.Component {
	return []components.Component{
		&ListPipelines{},
		&RunWorkflow{},
	}
}

func (s *Semaphore) Triggers() []triggers.Trigger {
	return []triggers.Trigger{}
}
