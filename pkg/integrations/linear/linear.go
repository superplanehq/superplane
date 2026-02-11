package linear

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("linear", &Linear{}, &LinearWebhookHandler{})
}

type Linear struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Teams  []Team  `json:"teams"`
	Labels []Label `json:"labels"`
}

func (l *Linear) Name() string {
	return "linear"
}

func (l *Linear) Label() string {
	return "Linear"
}

func (l *Linear) Icon() string {
	return "linear"
}

func (l *Linear) Description() string {
	return "Manage and react to issues in Linear"
}

func (l *Linear) Instructions() string {
	return ""
}

func (l *Linear) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Personal API key from Linear (Settings → API)",
		},
	}
}

func (l *Linear) Components() []core.Component {
	return []core.Component{
		&CreateIssue{},
	}
}

func (l *Linear) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueCreated{},
	}
}

func (l *Linear) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Linear) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}
	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	_, err = client.GetViewer()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("error listing teams: %w", err)
	}

	labels, err := client.ListLabels()
	if err != nil {
		return fmt.Errorf("error listing labels: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{Teams: teams, Labels: labels})
	ctx.Integration.Ready()
	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {}

func (l *Linear) Actions() []core.Action {
	return []core.Action{}
}

func (l *Linear) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
