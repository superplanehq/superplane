package cloudsmith

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	TagActionAdd     = "Add"
	TagActionClear   = "Clear"
	TagActionReplace = "Replace"
	TagActionRemove  = "Remove"
)

type TagPackage struct{}

type TagPackageSpec struct {
	PackageSpec `mapstructure:",squash"`
	Action      string   `json:"action" mapstructure:"action"`
	IsImmutable bool     `json:"isImmutable" mapstructure:"isImmutable"`
	Tags        []string `json:"tags" mapstructure:"tags"`
}

func (t *TagPackage) Name() string {
	return "cloudsmith.tagPackage"
}

func (t *TagPackage) Label() string {
	return "Tag Package"
}

func (t *TagPackage) Description() string {
	return "Add, replace, clear, or remove tags for a Cloudsmith package"
}

func (t *TagPackage) Documentation() string {
	return `The Tag Package component updates tags for a Cloudsmith package.

## Configuration

- **Repository**: The repository that contains the package.
- **Package Identifier**: The Cloudsmith package identifier (` + "`slug_perm`" + `).
- **Action**: ` + "`Add`" + `, ` + "`Clear`" + `, ` + "`Replace`" + `, or ` + "`Remove`" + `.
- **Tags**: Tags to apply. Not required when the action is ` + "`Clear`" + `.
- **Immutable**: Marks newly created tags immutable when supported by Cloudsmith.

## Output

Emits the updated package returned by Cloudsmith on the default channel.`
}

func (t *TagPackage) Icon() string {
	return "tag"
}

func (t *TagPackage) Color() string {
	return "gray"
}

func (t *TagPackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *TagPackage) Configuration() []configuration.Field {
	return packageConfigurationFields(
		configuration.Field{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How Cloudsmith should update the package tags",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Add", Value: TagActionAdd},
						{Label: "Clear", Value: TagActionClear},
						{Label: "Replace", Value: TagActionReplace},
						{Label: "Remove", Value: TagActionRemove},
					},
				},
			},
		},
		configuration.Field{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Description: "Tags to apply. Not required for Clear.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "action", Values: []string{TagActionAdd, TagActionReplace, TagActionRemove}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		configuration.Field{
			Name:        "isImmutable",
			Label:       "Immutable",
			Type:        configuration.FieldTypeBool,
			Description: "Create tags as immutable when adding or replacing tags",
		},
	)
}

func (t *TagPackage) Setup(ctx core.SetupContext) error {
	_, err := decodeTagPackageSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return setupPackageComponent(ctx)
}

func (t *TagPackage) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeTagPackageSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	owner, repository, identifier, err := packageRequestParts(spec.PackageSpec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	pkg, err := client.TagPackage(owner, repository, identifier, PackageTagRequest{
		Action:      spec.Action,
		IsImmutable: spec.IsImmutable,
		Tags:        spec.Tags,
	})
	if err != nil {
		return fmt.Errorf("failed to tag package: %v", err)
	}
	if pkg == nil {
		return fmt.Errorf("tag returned empty response")
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PackageTaggedPayloadType, []any{pkg})
}

func (t *TagPackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TagPackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueueItem(ctx)
}

func (t *TagPackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return defaultHandleWebhook(ctx)
}

func (t *TagPackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (t *TagPackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *TagPackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func decodeTagPackageSpec(configuration any) (TagPackageSpec, error) {
	spec := TagPackageSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return TagPackageSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	packageSpec, err := decodePackageSpec(configuration)
	if err != nil {
		return TagPackageSpec{}, err
	}

	spec.PackageSpec = packageSpec
	spec.Action = strings.TrimSpace(spec.Action)
	if !isValidTagAction(spec.Action) {
		return TagPackageSpec{}, fmt.Errorf("action must be one of %s, %s, %s, or %s", TagActionAdd, TagActionClear, TagActionReplace, TagActionRemove)
	}

	if spec.Action == TagActionClear {
		spec.Tags = nil
	} else {
		spec.Tags = cleanTags(spec.Tags)
		if len(spec.Tags) == 0 {
			return TagPackageSpec{}, errors.New("tags are required unless action is Clear")
		}
	}

	return spec, nil
}

func isValidTagAction(action string) bool {
	switch action {
	case TagActionAdd, TagActionClear, TagActionReplace, TagActionRemove:
		return true
	default:
		return false
	}
}

func cleanTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}

	return cleaned
}
