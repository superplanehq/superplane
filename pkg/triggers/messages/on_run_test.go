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

func TestOnRun_HandleHook_AcceptsPlanningSessionWithoutEvent(t *testing.T) {
	trigger := &OnRun{}
	events := &contexts.EventContext{}

	_, err := trigger.HandleHook(core.TriggerHookContext{
		Name: "onMessage",
		Parameters: map[string]any{
			"planning_session": map[string]any{
				"factory_id": "factory-1",
				"repository": "acme/payments",
			},
		},
		Events: events,
	})
	require.NoError(t, err)
	assert.Empty(t, events.Payloads)
}
