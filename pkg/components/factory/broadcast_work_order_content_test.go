package factory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestBroadcastWorkOrderContent_ValidatesConfiguration(t *testing.T) {
	component := &BroadcastWorkOrderContent{}
	fields := component.Configuration()

	t.Run("rejects a missing summary", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"url":     "https://preview.example.com",
		})
		require.Error(t, err)
	})

	t.Run("accepts a summary and a link", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"summary":  "Preview environment is ready",
			"url":      "https://preview.example.com",
			"urlLabel": "Preview",
		})
		require.NoError(t, err)
	})
}

func TestBroadcastWorkOrderContent_Execute(t *testing.T) {
	component := &BroadcastWorkOrderContent{}

	t.Run("emits the broadcast after the activity log write", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "order-1",
				"summary":  "Preview environment is ready",
				"body":     "Open the preview environment.",
				"url":      "https://preview.example.com",
				"urlLabel": "Preview",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.broadcastCalls)
		assert.Equal(t, core.BroadcastWorkOrderContentParams{
			OrderID:  "order-1",
			Summary:  "Preview environment is ready",
			Body:     "Open the preview environment.",
			URL:      "https://preview.example.com",
			URLLabel: "Preview",
		}, factoryCtx.broadcastParams)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "workOrder.contentBroadcast", stateCtx.Type)
	})

	t.Run("does not emit when the activity log write fails", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{broadcastErr: errors.New("link URL must be an absolute http or https URL")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId": "order-1",
				"summary": "Preview environment is ready",
				"url":     "javascript:alert(1)",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Empty(t, stateCtx.Channel)
	})
}
