package canvasdata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get.json
var getExampleOutputBytes []byte

var getExampleOutputOnce sync.Once
var getExampleOutput map[string]any

const GetComponentName = "canvasdata.get"
const GetPayloadType = "canvasdata.get.finished"

func init() {
	registry.RegisterComponent(GetComponentName, &GetCanvasData{})
}

// VersionOption values: "current", "previous", or "2" for two steps back, etc.
type GetSpec struct {
	Key             string `json:"key"`
	VersionOption   string `json:"versionOption"`
	VersionStepsBack *int   `json:"versionStepsBack,omitempty"`
}

type GetCanvasData struct{}

func (c *GetCanvasData) Name() string {
	return GetComponentName
}

func (c *GetCanvasData) Label() string {
	return "Get Canvas Data"
}

func (c *GetCanvasData) Description() string {
	return "Read a value from the canvas data store, optionally a previous version."
}

func (c *GetCanvasData) Documentation() string {
	return `Get Canvas Data reads a value for a key from the canvas data store. You can read the current (latest) value or a previous version for rollback or comparison.

## Use Cases

- **Rollback**: Get the previous version of a key (e.g. app/backend/last_version) to roll back.
- **Compare**: Read current and previous in separate nodes to detect changes.
- **Ephemeral teardown**: Read the list of machine IDs stored by an earlier step.

## Version

- **Current**: Latest value written for this key.
- **Previous**: One step back in history.
- **N steps back**: Specify how many versions back (e.g. 2 for two steps back).

If no value exists for the key (or the requested offset is beyond history), the component emits with found false and no value.`
}

func (c *GetCanvasData) Icon() string {
	return "database-search"
}

func (c *GetCanvasData) Color() string {
	return "blue"
}

func (c *GetCanvasData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCanvasData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&getExampleOutputOnce, getExampleOutputBytes, &getExampleOutput)
}

func (c *GetCanvasData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "app/my-service/last_version",
			Description: "Canvas-scoped key to read.",
		},
		{
			Name:        "versionOption",
			Label:       "Version",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Which version to read: current (latest), previous, or N steps back.",
			Default:     "current",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Current (latest)", Value: "current"},
						{Label: "Previous", Value: "previous"},
						{Label: "N steps back", Value: "steps_back"},
					},
				},
			},
		},
		{
			Name:        "versionStepsBack",
			Label:       "Steps back",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Number of versions back (1 = previous, 2 = two steps back, etc.).",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "versionOption", Values: []string{"steps_back"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "versionOption", Values: []string{"steps_back"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { n := 1; return &n }(),
					Max: func() *int { n := 100; return &n }(),
				},
			},
			Default: "1",
		},
	}
}

func (c *GetCanvasData) Execute(ctx core.ExecutionContext) error {
	spec := GetSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Key == "" {
		return fmt.Errorf("key is required")
	}

	offset := 0
	switch spec.VersionOption {
	case "", "current":
		offset = 0
	case "previous":
		offset = 1
	case "steps_back":
		if spec.VersionStepsBack != nil && *spec.VersionStepsBack >= 1 {
			offset = *spec.VersionStepsBack
		} else {
			offset = 1
		}
	default:
		offset = 0
	}

	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return fmt.Errorf("invalid workflow id: %w", err)
	}

	rec, err := models.GetCanvasData(canvasID, spec.Key, offset)
	if err != nil {
		return fmt.Errorf("get canvas data: %w", err)
	}

	payload := map[string]any{
		"key":   spec.Key,
		"found": rec != nil,
	}
	if rec != nil {
		payload["value"] = parseValue(rec.Value)
		payload["created_at"] = ""
		if rec.CreatedAt != nil {
			payload["created_at"] = rec.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetPayloadType,
		[]any{payload},
	)
}

// parseValue returns the value as string or parsed JSON object/array for downstream use.
func parseValue(s string) any {
	if s == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return s
}

func (c *GetCanvasData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCanvasData) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetCanvasData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("get canvas data does not support actions")
}

func (c *GetCanvasData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCanvasData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCanvasData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetCanvasData) Cleanup(ctx core.SetupContext) error {
	return nil
}
