package getdata

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "getData"
const PayloadType = "data.get"
const (
	channelFound    = "found"
	channelNotFound = "notFound"
)

func init() {
	registry.RegisterComponent(ComponentName, &GetData{})
}

type GetData struct{}

type Spec struct {
	Key         string  `json:"key"`
	Mode        string  `json:"mode"`
	MatchBy     *string `json:"matchBy,omitempty"`
	MatchValue  any     `json:"matchValue,omitempty"`
	ReturnField *string `json:"returnField,omitempty"`
}

func (c *GetData) Name() string {
	return ComponentName
}

func (c *GetData) Label() string {
	return "Get Data"
}

func (c *GetData) Description() string {
	return "Get a key/value pair from canvas storage"
}

func (c *GetData) Documentation() string {
	return `The Get Data component reads a value from canvas-level storage by key.

## Use Cases

- **Reuse shared state**: Read values previously stored with Set Data
- **Cross-path lookups**: Access values set in another branch
- **Conditional logic**: Fetch stored flags and IDs for downstream decisions

## How It Works

1. Reads ` + "`key`" + ` from configuration
2. Optionally filters list values by a field and value
3. Emits a ` + "`data.get`" + ` event on either ` + "`found`" + ` or ` + "`notFound`" + ` with ` + "`key`" + `, ` + "`value`" + `, and ` + "`exists`" + ``
}

func (c *GetData) Icon() string {
	return "database-zap"
}

func (c *GetData) Color() string {
	return "blue"
}

func (c *GetData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelFound, Label: "Found"},
		{Name: channelNotFound, Label: "Not Found"},
	}
}

func (c *GetData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Description: "Canvas data key to fetch",
			Required:    true,
		},
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Description: "How to read this key",
			Required:    true,
			Default:     "value",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Read whole value", Value: "value"},
						{Label: "Find item in list", Value: "listLookup"},
					},
				},
			},
		},
		{
			Name:        "matchBy",
			Label:       "Match By",
			Type:        configuration.FieldTypeString,
			Description: "Field to match in list items",
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"listLookup"}},
			},
		},
		{
			Name:        "matchValue",
			Label:       "Match Value",
			Type:        configuration.FieldTypeExpression,
			Description: "Value to match against selected field",
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"listLookup"}},
			},
		},
		{
			Name:        "returnField",
			Label:       "Return Field (optional)",
			Type:        configuration.FieldTypeString,
			Description: "Field to return from the matched item (leave empty to return full item)",
			Required:    false,
			Togglable:   true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"listLookup"}},
			},
		},
	}
}

func (c *GetData) Execute(ctx core.ExecutionContext) error {
	if ctx.CanvasData == nil {
		return fmt.Errorf("canvas data context is not available")
	}

	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Key = strings.TrimSpace(spec.Key)
	if spec.Key == "" {
		return fmt.Errorf("key is required")
	}
	if spec.Mode == "" {
		spec.Mode = "value"
	}

	value, exists, err := ctx.CanvasData.Get(spec.Key)
	if err != nil {
		return fmt.Errorf("failed to get canvas data: %w", err)
	}

	selectedValue := value
	selectedExists := exists
	if spec.Mode == "listLookup" {
		selectedValue, selectedExists, err = lookupListValue(spec, value, exists)
		if err != nil {
			return err
		}
	}

	channel := channelNotFound
	if selectedExists {
		channel = channelFound
	}

	return ctx.ExecutionState.Emit(
		channel,
		PayloadType,
		[]any{
			map[string]any{
				"key":    spec.Key,
				"value":  selectedValue,
				"exists": selectedExists,
			},
		},
	)
}

func lookupListValue(spec Spec, sourceValue any, sourceExists bool) (any, bool, error) {
	if !sourceExists {
		return nil, false, nil
	}

	matchBy := ""
	if spec.MatchBy != nil {
		matchBy = strings.TrimSpace(*spec.MatchBy)
	}
	if matchBy == "" {
		return nil, false, fmt.Errorf("matchBy is required for listLookup mode")
	}

	items, ok := sourceValue.([]any)
	if !ok {
		return nil, false, fmt.Errorf("key %s is not a list", spec.Key)
	}

	for _, item := range items {
		objectItem, ok := item.(map[string]any)
		if !ok {
			continue
		}

		candidate, ok := objectItem[matchBy]
		if !ok {
			continue
		}

		if fmt.Sprintf("%v", candidate) != fmt.Sprintf("%v", spec.MatchValue) {
			continue
		}

		returnField := ""
		if spec.ReturnField != nil {
			returnField = strings.TrimSpace(*spec.ReturnField)
		}
		if returnField == "" {
			return objectItem, true, nil
		}

		fieldValue, ok := objectItem[returnField]
		if !ok {
			return nil, false, nil
		}

		return fieldValue, true, nil
	}

	return nil, false, nil
}

func (c *GetData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetData) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("getData does not support actions")
}

func (c *GetData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetData) Cleanup(ctx core.SetupContext) error {
	return nil
}
