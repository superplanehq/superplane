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
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	Teams  []Team  `json:"teams"`
	UserID string  `json:"userId"`
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
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
	return `To connect Linear to SuperPlane:

1. Go to Linear Settings → API → Personal API Keys
2. Create a new API key with read and write permissions
3. Copy the API key and paste it below

Note: Linear uses GraphQL for its API. This integration will query teams and issues automatically.`
}

func (l *Linear) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Linear Personal API Key (from Settings → API)",
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
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client := NewClient(ctx.HTTP, ctx.Integration)

	// Verify credentials and get user info
	user, err := client.GetViewer()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	// Fetch teams
	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("error listing teams: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		Teams:  teams,
		UserID: user.ID,
	})
	ctx.Integration.Ready()
	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {
	handler := &LinearWebhookHandler{}
	handler.HandleWebhook(ctx)
}

func (l *Linear) Actions() []core.Action {
	return []core.Action{}
}

func (l *Linear) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
