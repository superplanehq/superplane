package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddWorkOrderArtifactComponentName = "addWorkOrderArtifact"

func init() {
	registry.RegisterAction(AddWorkOrderArtifactComponentName, &AddWorkOrderArtifact{})
}

type AddWorkOrderArtifact struct{}

// ArtifactDataEntry is one {name, value} row in the free-form data
// list; flattened into `map[string]any` at Execute time.
type ArtifactDataEntry struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type AddWorkOrderArtifactConfiguration struct {
	ArtifactType string              `json:"artifactType" mapstructure:"artifactType"`
	URL          string              `json:"url" mapstructure:"url"`
	Number       string              `json:"number" mapstructure:"number"`
	Title        string              `json:"title" mapstructure:"title"`
	Body         string              `json:"body" mapstructure:"body"`
	Data         []ArtifactDataEntry `json:"data" mapstructure:"data"`
}

func (c *AddWorkOrderArtifact) Name() string {
	return AddWorkOrderArtifactComponentName
}

func (c *AddWorkOrderArtifact) Label() string {
	return "Add Work Order Artifact"
}

func (c *AddWorkOrderArtifact) Description() string {
	return "Attach a typed artifact (PR or markdown) to a work order"
}

func (c *AddWorkOrderArtifact) Documentation() string {
	return `The Add Work Order Artifact component stores a typed artifact against a work order.

Supported types:

- **Pull request** (` + "`pr`" + `): requires ` + "`url`" + `; optional ` + "`number`" + ` and ` + "`title`" + `.
- **Markdown note** (` + "`markdown`" + `): requires ` + "`body`" + `; optional ` + "`title`" + `.

Both types accept a free-form ` + "`data`" + ` list of ` + "`{name, value}`" + ` entries that gets merged into the artifact's ` + "`data`" + ` map. Typed inputs take precedence over free-form entries with the same key. This component can only be used in factory-owned apps.`
}

func (c *AddWorkOrderArtifact) Icon() string {
	return "factory"
}

func (c *AddWorkOrderArtifact) Color() string {
	return "blue"
}

func (c *AddWorkOrderArtifact) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.artifactAdded",
		"data": map[string]any{
			"artifact": map[string]any{
				"id":   "art-123",
				"type": "pr",
				"data": map[string]any{
					"url":      "https://github.com/example/repo/pull/42",
					"number":   "42",
					"title":    "Draft implementation",
					"provider": "github",
				},
			},
		},
	}
}

func (c *AddWorkOrderArtifact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddWorkOrderArtifact) Configuration() []configuration.Field {
	prOnly := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr"}}}
	markdownOnly := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"markdown"}}}
	bothTypes := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown"}}}

	return []configuration.Field{
		{
			Name:        "artifactType",
			Label:       "Type",
			Description: "The kind of artifact to store",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "pr",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Pull Request", Value: "pr"},
						{Label: "Markdown", Value: "markdown"},
					},
				},
			},
		},
		{
			Name:                 "url",
			Label:                "URL",
			Description:          "Link to the pull request (must be http or https)",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: prOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"pr"}},
			},
		},
		{
			Name:                 "number",
			Label:                "Number",
			Description:          "Optional pull request number (rendered as #<n>)",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: prOnly,
		},
		{
			Name:                 "body",
			Label:                "Body",
			Description:          "Markdown note body — rendered inline in the work order timeline",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			VisibilityConditions: markdownOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"markdown"}},
			},
		},
		{
			Name:                 "title",
			Label:                "Title",
			Description:          "Optional artifact title",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: bothTypes,
		},
		{
			Name:                 "data",
			Label:                "Metadata",
			Description:          "Extra name/value pairs merged into the artifact's data map (typed fields above take precedence on name collisions)",
			Type:                 configuration.FieldTypeList,
			Required:             false,
			VisibilityConditions: bothTypes,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Entry",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
				},
			},
		},
	}
}

func (c *AddWorkOrderArtifact) Execute(ctx core.ExecutionContext) error {
	config := AddWorkOrderArtifactConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	data := buildArtifactData(config)

	artifact, err := ctx.Factory.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type: config.ArtifactType,
		Data: data,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.artifactAdded",
		[]any{map[string]any{
			"artifact": artifact,
		}},
	)
}

func (c *AddWorkOrderArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddWorkOrderArtifact) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddWorkOrderArtifact) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddWorkOrderArtifact) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// buildArtifactData folds the free-form list into a map and layers the
// typed inputs on top, so a user who defines both `url` and a `url`
// row still ends up with the typed value on the wire.
func buildArtifactData(config AddWorkOrderArtifactConfiguration) map[string]any {
	data := artifactDataToMap(config.Data)

	typed := map[string]string{
		"url":    config.URL,
		"number": config.Number,
		"title":  config.Title,
		"body":   config.Body,
	}

	for key, value := range typed {
		if value == "" {
			continue
		}
		if data == nil {
			data = map[string]any{}
		}
		data[key] = value
	}

	return data
}

// artifactDataToMap flattens the list into a map; blank names are
// skipped and duplicate names take the last value written.
func artifactDataToMap(entries []ArtifactDataEntry) map[string]any {
	if len(entries) == 0 {
		return nil
	}

	result := make(map[string]any, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		result[entry.Name] = entry.Value
	}

	return result
}
