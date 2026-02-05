package dockerhub

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DescribeImageTag struct{}

type DescribeImageTagSpec struct {
	Namespace  string `json:"namespace" mapstructure:"namespace"`
	Repository string `json:"repository" mapstructure:"repository"`
	Tag        string `json:"tag" mapstructure:"tag"`
}

func (d *DescribeImageTag) Name() string {
	return "dockerhub.describeImageTag"
}

func (d *DescribeImageTag) Label() string {
	return "Describe Image Tag"
}

func (d *DescribeImageTag) Description() string {
	return "Get details about a specific Docker Hub image tag"
}

func (d *DescribeImageTag) Icon() string {
	return "docker"
}

func (d *DescribeImageTag) Color() string {
	return "gray"
}

func (d *DescribeImageTag) Documentation() string {
	return `The Describe Image Tag component retrieves detailed information about a specific Docker Hub image tag.

## Use Cases

- **Deployment verification**: Verify an image tag exists before deployment
- **Get image digest**: Retrieve the exact digest for immutable deployments
- **Audit image metadata**: Get size, platforms, and last update information

## Configuration

- **Namespace**: The Docker Hub namespace (username or organization)
- **Repository**: Select the Docker Hub repository
- **Tag**: The specific tag to describe (e.g., ` + "`latest`" + `, ` + "`v1.0.0`" + `)

## Outputs

The component emits tag details including:
- ` + "`name`" + `: The tag name
- ` + "`last_updated`" + `: When the tag was last updated
- ` + "`full_size`" + `: The full size of the image in bytes
- ` + "`digest`" + `: The image digest
- ` + "`images`" + `: Array of platform-specific images with architecture and OS info
`
}

func (d *DescribeImageTag) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DescribeImageTag) Configuration() []configuration.Field {
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
			Description: "Select the Docker Hub repository",
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "latest",
			Description: "The image tag to describe",
			Placeholder: "latest",
		},
	}
}

func (d *DescribeImageTag) Setup(ctx core.SetupContext) error {
	spec := DescribeImageTagSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Namespace == "" {
		return errors.New("namespace is required")
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	if spec.Tag == "" {
		return errors.New("tag is required")
	}

	return nil
}

func (d *DescribeImageTag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DescribeImageTag) Execute(ctx core.ExecutionContext) error {
	spec := DescribeImageTagSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	tag, err := client.GetTag(spec.Namespace, spec.Repository, spec.Tag)
	if err != nil {
		return ctx.ExecutionState.Fail("not_found", fmt.Sprintf("error getting tag: %v", err))
	}

	// Convert response to map for emission
	output := map[string]any{
		"name":         tag.Name,
		"full_size":    tag.FullSize,
		"last_updated": tag.LastUpdated,
		"digest":       tag.Digest,
		"media_type":   tag.MediaType,
		"images":       tag.Images,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "dockerhub.tag", []any{output})
}

func (d *DescribeImageTag) Actions() []core.Action {
	return []core.Action{}
}

func (d *DescribeImageTag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DescribeImageTag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (d *DescribeImageTag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DescribeImageTag) Cleanup(ctx core.SetupContext) error {
	return nil
}
