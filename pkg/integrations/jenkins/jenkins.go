package jenkins

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("jenkins", &Jenkins{})
}

type Jenkins struct{}

type Configuration struct {
	URL      string `json:"url"`
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
	return ""
}

func (j *Jenkins) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jenkins server URL",
			Placeholder: "e.g. https://jenkins.example.com",
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jenkins username",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Jenkins API token",
		},
	}
}

func (j *Jenkins) Components() []core.Component {
	return []core.Component{
		&TriggerBuild{},
	}
}

func (j *Jenkins) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (j *Jenkins) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.URL == "" {
		return fmt.Errorf("url is required")
	}

	if config.Username == "" {
		return fmt.Errorf("username is required")
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	_, err = client.GetServerInfo()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (j *Jenkins) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jenkins) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (j *Jenkins) Actions() []core.Action {
	return []core.Action{}
}

func (j *Jenkins) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (j *Jenkins) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "job" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	jobs, err := client.ListJobs()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(jobs))
	for _, job := range jobs {
		if job.FullName == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: job.FullName,
			ID:   job.FullName,
		})
	}

	return resources, nil
}
