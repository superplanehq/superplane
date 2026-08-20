package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "createWorkOrder"

func init() {
	registry.RegisterAction(ComponentName, &CreateWorkOrder{})
}

type CreateWorkOrder struct{}

type CreateWorkOrderArtifactDataEntry struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type CreateWorkOrderConfiguration struct {
	Title           string                           `json:"title" mapstructure:"title"`
	Description     string                           `json:"description" mapstructure:"description"`
	ArtifactType    string                           `json:"artifactType" mapstructure:"artifactType"`
	ArtifactURL     string                           `json:"artifactUrl" mapstructure:"artifactUrl"`
	ArtifactTitle   string                           `json:"artifactTitle" mapstructure:"artifactTitle"`
	ArtifactKey     string                           `json:"artifactKey" mapstructure:"artifactKey"`
	ArtifactData    []CreateWorkOrderArtifactDataEntry `json:"artifactData" mapstructure:"artifactData"`
}

func (c *CreateWorkOrder) Name() string {
	return ComponentName
}

func (c *CreateWorkOrder) Label() string {
	return "Create Work Order"
}

func (c *CreateWorkOrder) Description() string {
	return "Create a new work order"
}

func (c *CreateWorkOrder) Documentation() string {
	return `The Create Work Order component creates a new work order in the factory.

When you set the optional artifact fields, it also attaches the artifact in the same transaction. This is useful for event-driven intake flows that need an idempotency key such as a source issue URL.

This component can only be used in factory-owned apps.`
}

func (c *CreateWorkOrder) Icon() string {
	return "factory"
}

func (c *CreateWorkOrder) Color() string {
	return "blue"
}

func (c *CreateWorkOrder) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.created",
		"data": map[string]any{
			"workOrder": map[string]any{
				"id":          "123",
				"title":       "Work Order 1",
				"description": "Work Order 1 description",
			},
		},
	}
}

func (c *CreateWorkOrder) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateWorkOrder) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Title",
			Description: "The title of the work order",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "description",
			Label:       "Description",
			Description: "The description of the work order",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "artifactType",
			Label:       "Initial Artifact Type",
			Description: "Optional artifact to attach as part of work order creation",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Link", Value: "link"},
					},
				},
			},
		},
		{
			Name:                 "artifactUrl",
			Label:                "Initial Artifact URL",
			Description:          "HTTP or HTTPS URL to attach when the initial artifact type is link",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "artifactType", Values: []string{"link"}}},
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"link"}}},
		},
		{
			Name:                 "artifactTitle",
			Label:                "Initial Artifact Title",
			Description:          "Optional chip label for the attached link artifact",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"link"}}},
		},
		{
			Name:        "artifactKey",
			Label:       "Initial Artifact Key",
			Description: "Optional stable key used to find an existing work order before creation",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:                 "artifactData",
			Label:                "Initial Artifact Metadata",
			Description:          "Extra name/value pairs merged into the attached artifact data",
			Type:                 configuration.FieldTypeList,
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"link"}}},
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

func (c *CreateWorkOrder) Execute(ctx core.ExecutionContext) error {
	config := CreateWorkOrderConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	params := core.WorkOrderParams{
		Title:       config.Title,
		Description: config.Description,
	}
	if config.ArtifactType != "" {
		artifactData := map[string]any{}
		for _, entry := range config.ArtifactData {
			if entry.Name == "" {
				continue
			}
			artifactData[entry.Name] = entry.Value
		}
		if config.ArtifactURL != "" {
			artifactData["url"] = config.ArtifactURL
		}
		if config.ArtifactTitle != "" {
			artifactData["title"] = config.ArtifactTitle
		}
		params.Artifact = &core.WorkOrderArtifactSeed{
			Type: config.ArtifactType,
			Data: artifactData,
			Key:  config.ArtifactKey,
		}
	}

	workOrder, created, err := ctx.Factory.CreateWorkOrder(params)
	if err != nil {
		return err
	}
	if !created {
		return ctx.ExecutionState.Pass()
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.created",
		[]any{map[string]any{
			"workOrder": workOrder,
		}},
	)
}

func (c *CreateWorkOrder) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkOrder) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateWorkOrder) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateWorkOrder) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkOrder) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateWorkOrder) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
