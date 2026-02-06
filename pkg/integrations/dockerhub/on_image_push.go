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
	Namespace  string                    `json:"namespace" mapstructure:"namespace"`
	Repository string                    `json:"repository" mapstructure:"repository"`
	Tags       []configuration.Predicate `json:"tags" mapstructure:"tags"`
}

type OnImagePushMetadata struct {
	Repository *RepositoryMetadata `json:"repository,omitempty" mapstructure:"repository"`
	WebhookURL string              `json:"webhookUrl,omitempty" mapstructure:"webhookUrl"`
}

type RepositoryMetadata struct {
	Namespace   string `json:"namespace" mapstructure:"namespace"`
	Name        string `json:"name" mapstructure:"name"`
	FullName    string `json:"fullName" mapstructure:"fullName"`
	URL         string `json:"url" mapstructure:"url"`
	Description string `json:"description,omitempty" mapstructure:"description"`
}

func (p *OnImagePush) Name() string {
	return "dockerhub.onImagePush"
}

func (p *OnImagePush) Label() string {
	return "On Image Push"
}

func (p *OnImagePush) Description() string {
	return "Listen to Docker Hub image push events"
}

func (p *OnImagePush) Documentation() string {
	return `The On Image Push trigger starts a workflow execution when an image is pushed to a Docker Hub repository.

## Use Cases

- **Deployment automation**: Automatically deploy when new images are pushed
- **Security scanning**: Trigger security scans on new image versions
- **Notification workflows**: Send notifications when images are published
- **CI/CD pipelines**: Chain workflows based on image updates

## Configuration

- **Namespace**: The Docker Hub namespace (username or organization)
- **Repository**: Select the Docker Hub repository to monitor
- **Tags**: Filter which tags should trigger the workflow (e.g., ` + "`latest`" + `, ` + "`v*`" + `)

## Webhook Setup

Since Docker Hub doesn't support programmatic webhook creation, you'll need to manually configure the webhook:

1. Copy the webhook URL shown in the trigger configuration
2. Go to your Docker Hub repository settings
3. Navigate to Webhooks and add a new webhook
4. Paste the SuperPlane webhook URL

## Event Data

Each push event includes:
- **repository**: Repository information (name, namespace, URL)
- **push_data**: Push details including tag and pusher
- **tag**: The image tag that was pushed`
}

func (p *OnImagePush) Icon() string {
	return "docker"
}

func (p *OnImagePush) Color() string {
	return "blue"
}

func (p *OnImagePush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Docker Hub namespace (username or organization)",
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
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
			Description: "Select the Docker Hub repository to monitor",
		},
		{
			Name:     "tags",
			Label:    "Tags",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": "*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
			Description: "Filter which image tags should trigger the workflow",
		},
	}
}

func (p *OnImagePush) Setup(ctx core.TriggerContext) error {
	var metadata OnImagePushMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnImagePushConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	// Check if already configured with same repository
	if metadata.Repository != nil &&
		metadata.Repository.Namespace == config.Namespace &&
		metadata.Repository.Name == config.Repository {
		return nil
	}

	// Verify repository exists in Docker Hub
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repo, err := client.GetRepository(config.Namespace, config.Repository)
	if err != nil {
		return fmt.Errorf("failed to find repository %s/%s: %w", config.Namespace, config.Repository, err)
	}

	// Setup webhook URL - Docker Hub webhooks are manually configured
	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	// Store metadata
	err = ctx.Metadata.Set(OnImagePushMetadata{
		Repository: &RepositoryMetadata{
			Namespace:   repo.Namespace,
			Name:        repo.Name,
			FullName:    fmt.Sprintf("%s/%s", repo.Namespace, repo.Name),
			URL:         fmt.Sprintf("https://hub.docker.com/r/%s/%s", repo.Namespace, repo.Name),
			Description: repo.Description,
		},
		WebhookURL: webhookURL,
	})
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnImagePushConfiguration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to parse configuration: %w", err)
	}

	var payload WebhookPayload
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Verify this is for the configured repository
	expectedRepo := fmt.Sprintf("%s/%s", config.Namespace, config.Repository)
	if payload.Repository.RepoName != expectedRepo {
		// Not for this repository, ignore
		return http.StatusOK, nil
	}

	// Check if tag matches the filter
	if len(config.Tags) > 0 {
		if !configuration.MatchesAnyPredicate(config.Tags, payload.PushData.Tag) {
			// Tag doesn't match filter
			return http.StatusOK, nil
		}
	}

	// Emit the event
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	err = ctx.Events.Emit("dockerhub.imagePush", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (p *OnImagePush) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// splitRepository splits a repository name into namespace and name
func splitRepository(repoName string) (namespace, name string) {
	parts := strings.SplitN(repoName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Default namespace is "library" for official images
	return "library", repoName
}
