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

// Data is a free-form list of `{name, value}` entries the component
// author fills in per invocation. We turn it into a `map[string]any`
// before handing it to the context so callers downstream still see
// familiar structured metadata; keeping the config as a list avoids
// baking a fixed PR schema into the component (see PR #6569 review).
type ArtifactDataEntry struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type AddWorkOrderArtifactConfiguration struct {
	ArtifactType string              `json:"artifactType" mapstructure:"artifactType"`
	URL          string              `json:"url" mapstructure:"url"`
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
	return `The Add Work Order Artifact component stores a typed artifact against a work order. Supported types today are: pull requests (URL is required) and markdown notes (an inline body is required). Free-form metadata can be attached as name/value pairs. This component can only be used in factory-owned apps.`
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
				"id":    "art-123",
				"type":  "pr",
				"url":   "https://github.com/example/repo/pull/42",
				"title": "Draft implementation",
				"data": map[string]any{
					"number":   "42",
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
	prVisibility := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr"}}}
	markdownVisibility := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"markdown"}}}

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
			Description:          "Link to the pull request",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: prVisibility,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"pr"}},
			},
		},
		{
			Name:                 "title",
			Label:                "Title",
			Description:          "Optional artifact title",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown"}}},
		},
		{
			Name:                 "body",
			Label:                "Body",
			Description:          "Markdown body stored inline with the artifact",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			VisibilityConditions: markdownVisibility,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"markdown"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "markdown",
				},
			},
		},
		{
			Name:        "data",
			Label:       "Metadata",
			Description: "Free-form name/value pairs stored alongside the artifact (e.g. PR number, provider, refs)",
			Type:        configuration.FieldTypeList,
			Required:    false,
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

	data := artifactDataToMap(config.Data)

	artifact, err := ctx.Factory.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type:  config.ArtifactType,
		URL:   config.URL,
		Title: config.Title,
		Body:  config.Body,
		Data:  data,
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

// artifactDataToMap flattens the free-form `{name, value}` entries the
// component author configured into the `map[string]any` shape the model
// layer expects. Duplicate names take the last value written, matching
// how the UI would render them in-order.
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
