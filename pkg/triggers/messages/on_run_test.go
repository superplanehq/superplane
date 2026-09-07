package messages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnRun_HandleHook_RequiresAppWorkOrderOrPlanningSession(t *testing.T) {
	trigger := &OnRun{}
	events := &contexts.EventContext{}

	_, err := trigger.HandleHook(core.TriggerHookContext{
		Name:       "onMessage",
		Parameters: map[string]any{},
		Events:     events,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app, work_order, or planning_session is required")
	assert.Empty(t, events.Payloads)
}

func TestOnRun_HandleHook_EmitsPlanningSessionEvent(t *testing.T) {
	trigger := &OnRun{}
	events := &contexts.EventContext{}
	planningSession := map[string]any{
		"factory_id": "factory-1",
		"repository": "acme/payments",
	}

	_, err := trigger.HandleHook(core.TriggerHookContext{
		Name: "onMessage",
		Parameters: map[string]any{
			"planning_session": planningSession,
		},
		Events: events,
	})
	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	assert.Equal(t, "planning.session", events.Payloads[0].Type)
	assert.Equal(t, map[string]any{"planning_session": planningSession}, events.Payloads[0].Data)
}
