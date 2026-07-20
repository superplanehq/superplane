package cloudsmith

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRepository struct{}

type GetRepositorySpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

func (g *GetRepository) Name() string {
	return "cloudsmith.getRepository"
}

func (g *GetRepository) Label() string {
	return "Get Repository"
}

func (g *GetRepository) Description() string {
	return "Fetch details of a Cloudsmith repository"
}

func (g *GetRepository) Documentation() string {
	return `The Get Repository component retrieves detailed information about a specific Cloudsmith repository.

## Use Cases

- **Status checks**: Verify a repository exists and is reachable before publishing or promoting packages
- **Information retrieval**: Read repository visibility, namespace, and configuration
- **Storage monitoring**: Track storage usage, package counts, and download metrics
- **Compliance checks**: Inspect quarantined or policy-violating package counts before downstream actions

## Configuration

- **Repository**: The repository to retrieve (required, supports expressions). The value is the repository identifier in the form ` + "`owner/repository`" + `.

## Output

Returns the repository object including:
- **name**: A descriptive name for the repository
- **slug**: The slug that identifies the repository in URIs
- **namespace**: The namespace (owner) the repository belongs to
- **repository_type_str**: The visibility of the repository (Public, Private, Open-Source)
- **storage_region**: The Cloudsmith region in which package files are stored
- **size** / **size_str**: The calculated storage size of the repository
- **package_count**: The number of packages in the repository
- **num_downloads**: The number of downloads for packages in the repository
- **num_quarantined_packages**: The number of quarantined packages
- **num_policy_violated_packages**: The number of packages with policy violations`
}

func (g *GetRepository) Icon() string {
	return "info"
}

func (g *GetRepository) Color() string {
	return "gray"
}

func (g *GetRepository) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetRepository) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to retrieve",
			Placeholder: "Select repository",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (g *GetRepository) Setup(ctx core.SetupContext) error {
	spec := GetRepositorySpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	err = resolveRepositoryMetadata(ctx, spec.Repository)
	if err != nil {
		return fmt.Errorf("error resolving repository metadata: %v", err)
	}

	return nil
}

func (g *GetRepository) Execute(ctx core.ExecutionContext) error {
	spec := GetRepositorySpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	owner, identifier, err := parseRepositoryID(spec.Repository)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", spec.Repository, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	repository, err := client.GetRepository(owner, identifier)
	if err != nil {
		return fmt.Errorf("failed to get repository: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.repository.fetched",
		[]any{repository},
	)
}

func (g *GetRepository) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetRepository) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetRepository) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetRepository) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetRepository) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetRepository) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
