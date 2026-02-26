package cleardata

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

const ComponentName = "clearData"
const PayloadType = "data.clear"

const (
	channelCleared  = "cleared"
	channelNotFound = "notFound"
)

func init() {
	registry.RegisterComponent(ComponentName, &ClearData{})
}

type ClearData struct{}

type Spec struct {
	Key        string `json:"key"`
	MatchBy    string `json:"matchBy"`
	MatchValue any    `json:"matchValue"`
}

func (c *ClearData) Name() string {
	return ComponentName
}

func (c *ClearData) Label() string {
	return "Clear Data"
}

func (c *ClearData) Description() string {
	return "Remove matching entries from list data in canvas storage"
}

func (c *ClearData) Documentation() string {
	return `The Clear Data component removes matching items from list data in canvas-level storage.

## Use Cases

- **Cleanup mappings**: Remove PR->sandbox records after sandbox deletion
- **Cross-run lifecycle**: Keep shared list data in sync as resources are deleted
- **Data hygiene**: Remove stale entries from list-based canvas storage

## How It Works

1. Reads ` + "`key`" + `, ` + "`matchBy`" + `, and ` + "`matchValue`" + `
2. Finds list items where item[matchBy] == matchValue
3. Removes matching items and stores the updated list
4. Emits on ` + "`cleared`" + ` (removed at least one item) or ` + "`notFound`" + ``
}

func (c *ClearData) Icon() string {
	return "database-x"
}

func (c *ClearData) Color() string {
	return "orange"
}

func (c *ClearData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelCleared, Label: "Cleared"},
		{Name: channelNotFound, Label: "Not Found"},
	}
}

func (c *ClearData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Description: "Canvas data key to clear from",
			Required:    true,
		},
		{
			Name:        "matchBy",
			Label:       "Match By",
			Type:        configuration.FieldTypeString,
			Description: "Field name used to match list entries",
			Required:    true,
		},
		{
			Name:        "matchValue",
			Label:       "Match Value",
			Type:        configuration.FieldTypeExpression,
			Description: "Value to match against the selected field",
			Required:    true,
		},
	}
}

func (c *ClearData) Execute(ctx core.ExecutionContext) error {
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
	spec.MatchBy = strings.TrimSpace(spec.MatchBy)
	if spec.MatchBy == "" {
		return fmt.Errorf("matchBy is required")
	}

	value, exists, err := ctx.CanvasData.Get(spec.Key)
	if err != nil {
		return fmt.Errorf("failed to get canvas data: %w", err)
	}
	if !exists {
		return emitResult(ctx, channelNotFound, spec.Key, false, false, 0)
	}

	items, ok := value.([]any)
	if !ok {
		return fmt.Errorf("key %s is not a list", spec.Key)
	}

	filtered := make([]any, 0, len(items))
	removedCount := 0
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}

		candidate, ok := obj[spec.MatchBy]
		if !ok {
			filtered = append(filtered, item)
			continue
		}

		if fmt.Sprintf("%v", candidate) == fmt.Sprintf("%v", spec.MatchValue) {
			removedCount++
			continue
		}

		filtered = append(filtered, item)
	}

	if removedCount == 0 {
		return emitResult(ctx, channelNotFound, spec.Key, true, false, 0)
	}

	if err := ctx.CanvasData.Set(spec.Key, filtered); err != nil {
		return fmt.Errorf("failed to set canvas data: %w", err)
	}

	return emitResult(ctx, channelCleared, spec.Key, true, true, removedCount)
}

func emitResult(ctx core.ExecutionContext, channel string, key string, exists bool, removed bool, removedCount int) error {
	return ctx.ExecutionState.Emit(
		channel,
		PayloadType,
		[]any{
			map[string]any{
				"key":         key,
				"exists":      exists,
				"removed":     removed,
				"removedCount": removedCount,
			},
		},
	)
}

func (c *ClearData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ClearData) Actions() []core.Action {
	return []core.Action{}
}

func (c *ClearData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("clearData does not support actions")
}

func (c *ClearData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ClearData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ClearData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ClearData) Cleanup(ctx core.SetupContext) error {
	return nil
}
