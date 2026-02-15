package statuspage

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("statuspage", &Statuspage{})
}

type Statuspage struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
	PageID string `json:"pageId"`
}

type Metadata struct {
	Page *PageMetadata `json:"page,omitempty"`
}

type PageMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Statuspage) Name() string {
	return "statuspage"
}

func (s *Statuspage) Label() string {
	return "Statuspage"
}

func (s *Statuspage) Icon() string {
	return "activity"
}

func (s *Statuspage) Description() string {
	return "Manage incidents on Atlassian Statuspage"
}

func (s *Statuspage) Instructions() string {
	return `
1. **API Key:** Generate one in your [Statuspage account settings](https://manage.statuspage.io/login) under **API** section.
2. **Page ID:** Find it in your Statuspage dashboard URL or under **Page Settings**. Example: ` + "`abc123def456`" + `.
3. **Auth:** SuperPlane sends requests to [Statuspage API v1](https://developer.statuspage.io/) using the ` + "`Authorization: OAuth <API_KEY>`" + ` header.`
}

func (s *Statuspage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Statuspage API key",
		},
		{
			Name:        "pageId",
			Label:       "Page ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Statuspage page ID",
			Placeholder: "abc123def456",
		},
	}
}

func (s *Statuspage) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
		&UpdateIncident{},
		&GetIncident{},
	}
}

func (s *Statuspage) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (s *Statuspage) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Statuspage) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	if config.PageID == "" {
		return fmt.Errorf("pageId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	page, err := client.GetPage()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		Page: &PageMetadata{
			ID:   page.ID,
			Name: page.Name,
		},
	})

	ctx.Integration.Ready()
	return nil
}

func (s *Statuspage) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *Statuspage) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "component":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		components, err := client.ListComponents()
		if err != nil {
			return nil, fmt.Errorf("failed to list components: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(components))
		for _, c := range components {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: c.Name,
				ID:   c.ID,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (s *Statuspage) Actions() []core.Action {
	return []core.Action{}
}

func (s *Statuspage) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
