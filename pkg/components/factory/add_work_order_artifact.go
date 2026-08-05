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

type AddWorkOrderArtifactConfiguration struct {
	WorkOrderID  string         `json:"workOrderId" mapstructure:"workOrderId"`
	ArtifactType string         `json:"artifactType" mapstructure:"artifactType"`
	URL          string         `json:"url" mapstructure:"url"`
	Title        string         `json:"title" mapstructure:"title"`
	Body         string         `json:"body" mapstructure:"body"`
	Data         map[string]any `json:"data" mapstructure:"data"`
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
	return `The Add Work Order Artifact component stores a typed artifact against a work order. Supported types today are: pull requests (URL is required, plus optional rich metadata) and markdown notes (an inline body is required). This component can only be used in factory-owned apps.`
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
				"id":          "art-123",
				"workOrderId": "wo-123",
				"type":        "pr",
				"url":         "https://github.com/example/repo/pull/42",
				"title":       "Draft implementation",
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
			Name:        "workOrderId",
			Label:       "Work Order ID",
			Description: "The ID of the work order the artifact belongs to",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
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
			Name:                 "data",
			Label:                "Metadata",
			Description:          "Optional structured metadata for the artifact (e.g. PR number, provider, refs)",
			Type:                 configuration.FieldTypeObject,
			Required:             false,
			VisibilityConditions: prVisibility,
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{Name: "number", Label: "Number", Type: configuration.FieldTypeString},
						{Name: "repository", Label: "Repository", Type: configuration.FieldTypeString},
						{Name: "owner", Label: "Owner", Type: configuration.FieldTypeString},
						{Name: "provider", Label: "Provider", Type: configuration.FieldTypeString, Description: "e.g. github, gitlab"},
						{Name: "headRef", Label: "Head Ref", Type: configuration.FieldTypeString},
						{Name: "baseRef", Label: "Base Ref", Type: configuration.FieldTypeString},
						{Name: "state", Label: "State", Type: configuration.FieldTypeString, Description: "e.g. open, merged, closed"},
						{Name: "externalId", Label: "External ID", Type: configuration.FieldTypeString},
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

	artifact, err := ctx.Factory.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		WorkOrderID: config.WorkOrderID,
		Type:        config.ArtifactType,
		URL:         config.URL,
		Title:       config.Title,
		Body:        config.Body,
		Data:        config.Data,
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
