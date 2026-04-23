package manual

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestStart_Actions_DeclaresUserAccessibleRun(t *testing.T) {
	s := &Start{}
	actions := s.Actions()

	require.Len(t, actions, 1)
	action := actions[0]
	assert.Equal(t, ActionRun, action.Name)
	assert.True(t, action.UserAccessible)

	var paramNames []string
	var templateRequired bool
	var payloadRequired bool
	for _, param := range action.Parameters {
		paramNames = append(paramNames, param.Name)
		if param.Name == "template" {
			templateRequired = param.Required
		}
		if param.Name == "payload" {
			payloadRequired = param.Required
		}
	}

	assert.ElementsMatch(t, []string{"template", "payload"}, paramNames)
	assert.True(t, templateRequired, "template parameter must be required")
	assert.False(t, payloadRequired, "payload parameter must be optional")
}

func TestStart_HandleAction_EmitsWithConfiguredPayload(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{"message": "Hello, World!"}},
			map[string]any{"name": "Bye", "payload": map[string]any{"message": "Goodbye"}},
		},
	}

	result, err := s.HandleAction(core.TriggerActionContext{
		Name:          ActionRun,
		Parameters:    map[string]any{"template": "Hello"},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello", result["template"])

	require.Len(t, events.Payloads, 1)
	assert.Equal(t, "manual.run", events.Payloads[0].Type)
	payload, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, World!", payload["message"])
}

func TestStart_HandleAction_PayloadOverride(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{"message": "Hello, World!"}},
		},
	}

	_, err := s.HandleAction(core.TriggerActionContext{
		Name: ActionRun,
		Parameters: map[string]any{
			"template": "Hello",
			"payload":  map[string]any{"message": "Override"},
		},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	payload := events.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "Override", payload["message"])
}

func TestStart_HandleAction_UnknownTemplateListsAvailable(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{}},
			map[string]any{"name": "Bye", "payload": map[string]any{}},
		},
	}

	_, err := s.HandleAction(core.TriggerActionContext{
		Name:          ActionRun,
		Parameters:    map[string]any{"template": "Missing"},
		Configuration: config,
		Events:        events,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Missing")
	assert.Contains(t, err.Error(), "Hello")
	assert.Contains(t, err.Error(), "Bye")
	assert.Empty(t, events.Payloads)
}

func TestStart_HandleAction_RejectsMissingTemplate(t *testing.T) {
	s := &Start{}

	_, err := s.HandleAction(core.TriggerActionContext{
		Name:          ActionRun,
		Parameters:    map[string]any{},
		Configuration: map[string]any{},
		Events:        &contexts.EventContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template")
}

func TestStart_HandleAction_RejectsNoTemplatesConfigured(t *testing.T) {
	s := &Start{}

	_, err := s.HandleAction(core.TriggerActionContext{
		Name:          ActionRun,
		Parameters:    map[string]any{"template": "Hello"},
		Configuration: map[string]any{},
		Events:        &contexts.EventContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates configured")
}

func TestStart_HandleAction_RejectsUnknownAction(t *testing.T) {
	s := &Start{}

	_, err := s.HandleAction(core.TriggerActionContext{
		Name:   "nope",
		Events: &contexts.EventContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
