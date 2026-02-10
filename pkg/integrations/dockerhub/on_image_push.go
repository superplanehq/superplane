package dockerhub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnImagePush struct{}

type OnImagePushConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
}

type OnImagePushMetadata struct {
	Repository *RepositoryMetadata `json:"repository" mapstructure:"repository"`
	WebhookURL string              `json:"webhookUrl" mapstructure:"webhookUrl"`
}

type RepositoryMetadata struct {
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Name      string `json:"name" mapstructure:"name"`
}

type ImagePushPayload struct {
	CallbackURL string              `json:"callback_url"`
	PushData    ImagePushData       `json:"push_data"`
	Repository  ImagePushRepository `json:"repository"`
}

type ImagePushData struct {
	Tag      string `json:"tag"`
	PushedAt int64  `json:"pushed_at"`
	Pusher   string `json:"pusher"`
}

type ImagePushRepository struct {
	RepoName   string `json:"repo_name"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	RepoURL    string `json:"repo_url"`
	IsPrivate  bool   `json:"is_private"`
	Status     string `json:"status"`
	StarCount  int    `json:"star_count"`
	PullCount  int    `json:"pull_count"`
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
}

func (p *OnImagePush) Name() string {
	return "dockerhub.onImagePush"
}

func (p *OnImagePush) Label() string {
	return "On Image Push"
}

func (p *OnImagePush) Description() string {
	return "Listen to DockerHub image push events"
}

func (p *OnImagePush) Documentation() string {
	return `The On Image Push trigger starts a workflow execution when an image tag is pushed to DockerHub.

## Use Cases

- **Build pipelines**: Trigger builds and deployments on container pushes
- **Release workflows**: Promote artifacts when a new tag is published
- **Security automation**: Kick off scans or alerts for newly pushed images

## Configuration

- **Repository**: DockerHub repository name, in the format of ` + "`namespace/name`" + `
- **Tags**: Optional filters for image tags (for example: ` + "`latest`" + ` or ` + "`^v[0-9]+`" + `)

## Webhook Setup

This trigger generates a webhook URL in SuperPlane. Add that URL as a DockerHub webhook for the selected repository so DockerHub can deliver push events.`
}

func (p *OnImagePush) Icon() string {
	return "docker"
}

func (p *OnImagePush) Color() string {
	return "gray"
}

func (p *OnImagePush) ExampleData() map[string]any {
	return onImagePushExampleData()
}

func (p *OnImagePush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "dockerhub.repository",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "namespace",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "namespace",
							},
						},
					},
				},
			},
		},
		{
			Name:     "tags",
			Label:    "Tags",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnImagePush) Setup(ctx core.TriggerContext) error {
	metadata := OnImagePushMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnImagePushConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository := strings.TrimSpace(config.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in the format of namespace/name")
	}

	namespace := parts[0]
	repositoryName := parts[1]

	if metadata.Repository != nil &&
		metadata.Repository.Name == repositoryName &&
		metadata.Repository.Namespace == namespace &&
		metadata.WebhookURL != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repoInfo, err := client.GetRepository(namespace, repositoryName)
	if err != nil {
		return fmt.Errorf("failed to validate repository %s in namespace %s: %w", repositoryName, namespace, err)
	}

	webhookURL := metadata.WebhookURL
	if webhookURL == "" {
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}
	}

	return ctx.Metadata.Set(OnImagePushMetadata{
		WebhookURL: webhookURL,
		Repository: &RepositoryMetadata{
			Namespace: repoInfo.Namespace,
			Name:      repoInfo.Name,
		},
	})
}

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnImagePushConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnImagePushMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	payload := ImagePushPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	if metadata.Repository == nil {
		return http.StatusOK, nil
	}

	if metadata.Repository.Namespace != payload.Repository.Namespace {
		ctx.Logger.Infof("Ignoring event for namespace %s", payload.Repository.Namespace)
		return http.StatusOK, nil
	}

	if metadata.Repository.Name != payload.Repository.Name {
		ctx.Logger.Infof("Ignoring event for repository %s", payload.Repository.Name)
		return http.StatusOK, nil
	}

	if len(config.Tags) > 0 {
		tag := strings.TrimSpace(payload.PushData.Tag)
		if tag == "" {
			return http.StatusOK, nil
		}

		if !configuration.MatchesAnyPredicate(config.Tags, tag) {
			ctx.Logger.Infof("Ignoring event with non-matching tag %s", tag)
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit("dockerhub.image.push", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func (p *OnImagePush) Cleanup(ctx core.TriggerContext) error {
	return nil
}
