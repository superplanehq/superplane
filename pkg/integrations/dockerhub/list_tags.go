package dockerhub

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListTags struct{}

type ListTagsSpec struct {
	Repository string `json:"repository"`
	PageSize   int    `json:"pageSize"`
	NameFilter string `json:"nameFilter"`
}

func (l *ListTags) Name() string {
	return "dockerhub.listTags"
}

func (l *ListTags) Label() string {
	return "List Tags"
}

func (l *ListTags) Description() string {
	return "List tags for a Docker Hub repository"
}

func (l *ListTags) Icon() string {
	return "docker"
}

func (l *ListTags) Color() string {
	return "gray"
}

func (l *ListTags) Documentation() string {
	return `The List Tags component retrieves tags for a Docker Hub repository.

## Use Cases

- **Find latest tag**: Discover the most recent image version for deployment automation
- **Audit versions**: Review available versions before promoting an image
- **Cleanup automation**: List tags for retention policy enforcement

## Configuration

- **Repository**: The Docker Hub repository name (e.g., ` + "`library/nginx`" + ` or ` + "`myorg/myapp`" + `)
- **Page Size**: Number of results to return (optional, defaults to Docker Hub's default)
- **Name Filter**: Filter tags by name pattern (optional)

## Outputs

The component emits a list of tags, each containing:
- ` + "`name`" + `: The tag name
- ` + "`last_updated`" + `: When the tag was last updated
- ` + "`full_size`" + `: The full size of the image in bytes
- ` + "`digest`" + `: The image digest
- ` + "`images`" + `: Array of platform-specific images with architecture and OS info
`
}

func (l *ListTags) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListTags) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Docker Hub repository name (e.g., library/nginx or myorg/myapp)",
			Placeholder: "library/nginx",
		},
		{
			Name:        "pageSize",
			Label:       "Page Size",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Number of results per page (optional)",
		},
		{
			Name:        "nameFilter",
			Label:       "Name Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter tags by name pattern (optional)",
			Placeholder: "v1.*",
		},
	}
}

func (l *ListTags) Setup(ctx core.SetupContext) error {
	spec := ListTagsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	return nil
}

func (l *ListTags) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListTags) Execute(ctx core.ExecutionContext) error {
	spec := ListTagsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	response, err := client.ListTags(ListTagsRequest{
		Repository: spec.Repository,
		PageSize:   spec.PageSize,
		NameFilter: spec.NameFilter,
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error listing tags: %v", err))
	}

	// Convert response to map for emission
	output := map[string]any{
		"count":   response.Count,
		"results": response.Results,
	}

	if response.Next != "" {
		output["next"] = response.Next
	}
	if response.Previous != "" {
		output["previous"] = response.Previous
	}

	err = ctx.ExecutionState.Emit("default", "dockerhub.tags", []any{output})
	if err != nil {
		return fmt.Errorf("error emitting output: %v", err)
	}

	return nil
}

func (l *ListTags) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListTags) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListTags) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (l *ListTags) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListTags) Cleanup(ctx core.SetupContext) error {
	return nil
}
