package jenkins

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("jenkins", &Jenkins{})
}

type Jenkins struct{}

type Configuration struct {
	BaseURL  string `json:"baseUrl"`
	Username string `json:"username"`
	APIToken string `json:"apiToken"`
}

func (j *Jenkins) Name() string {
	return "jenkins"
}

func (j *Jenkins) Label() string {
	return "Jenkins"
}

func (j *Jenkins) Icon() string {
	return "jenkins"
}

func (j *Jenkins) Description() string {
	return "Trigger and monitor Jenkins builds"
}

func (j *Jenkins) Instructions() string {
	return `Create a Jenkins API Token in Jenkins → click your username (top right) → **Configure** → **API Token** → **Add new Token**.

Use your Jenkins **Base URL** (e.g. https://jenkins.example.com), your Jenkins **Username**, and the generated **API Token** below.`
}

func (j *Jenkins) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Jenkins server URL",
			Placeholder: "https://jenkins.example.com",
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Jenkins username",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Jenkins API Token",
		},
	}
}

func (j *Jenkins) Actions() []core.Action {
	return []core.Action{
		&TriggerBuild{},
		&GetBuild{},
	}
}

func (j *Jenkins) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (j *Jenkins) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jenkins) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	err = client.Verify()
	if err != nil {
		return fmt.Errorf("error verifying connection: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (j *Jenkins) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (j *Jenkins) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (j *Jenkins) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jenkins) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
