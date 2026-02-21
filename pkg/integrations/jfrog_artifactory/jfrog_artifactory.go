package jfrogartifactory

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("jfrogArtifactory", &JFrogArtifactory{}, &JFrogWebhookHandler{})
}

type JFrogArtifactory struct{}

func (j *JFrogArtifactory) Name() string {
	return "jfrogArtifactory"
}

func (j *JFrogArtifactory) Label() string {
	return "JFrog Artifactory"
}

func (j *JFrogArtifactory) Icon() string {
	return "jfrogArtifactory"
}

func (j *JFrogArtifactory) Description() string {
	return "Manage artifacts in JFrog Artifactory repositories"
}

func (j *JFrogArtifactory) Instructions() string {
	return `To set up the JFrog Artifactory integration:

1. Log in to your JFrog Platform
2. Go to **User Menu** (top right) -> **Edit Profile** -> **Authentication Settings**
3. Click **Generate an Identity Token**
4. Copy the token and paste it in the **Access Token** field below
5. Enter your JFrog Platform URL without the /artifactory suffix (e.g. https://mycompany.jfrog.io)`
}

func (j *JFrogArtifactory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "JFrog Platform URL (without /artifactory suffix)",
			Placeholder: "e.g. https://mycompany.jfrog.io",
		},
		{
			Name:        "accessToken",
			Label:       "Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "JFrog identity token or access token",
		},
	}
}

func (j *JFrogArtifactory) Components() []core.Component {
	return []core.Component{
		&GetArtifactInfo{},
		&DeleteArtifact{},
	}
}

func (j *JFrogArtifactory) Triggers() []core.Trigger {
	return []core.Trigger{&OnArtifactUploaded{}}
}

func (j *JFrogArtifactory) Sync(ctx core.SyncContext) error {
	rawURL, err := ctx.Integration.GetConfig("url")
	if err != nil {
		return fmt.Errorf("url is required")
	}

	if string(rawURL) == "" {
		return fmt.Errorf("url is required")
	}

	accessToken, err := ctx.Integration.GetConfig("accessToken")
	if err != nil {
		return fmt.Errorf("accessToken is required")
	}

	if string(accessToken) == "" {
		return fmt.Errorf("accessToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Ping(); err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (j *JFrogArtifactory) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *JFrogArtifactory) HandleRequest(ctx core.HTTPRequestContext) {
}

func (j *JFrogArtifactory) Actions() []core.Action {
	return []core.Action{}
}

func (j *JFrogArtifactory) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (j *JFrogArtifactory) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "repository" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	repos, err := client.ListRepositories()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(repos))
	for _, repo := range repos {
		if repo.Key == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repo.Key,
			ID:   repo.Key,
		})
	}

	return resources, nil
}
