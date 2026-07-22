package manual

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestStart_Hooks_DeclaresUserAccessibleRun(t *testing.T) {
	s := &Start{}
	hooks := s.Hooks()

	require.Len(t, hooks, 1)
	hook := hooks[0]
	assert.Equal(t, HookRun, hook.Name)
	assert.Equal(t, core.HookTypeUser, hook.Type)

	var paramNames []string
	var templateRequired bool
	for _, param := range hook.Parameters {
		paramNames = append(paramNames, param.Name)
		if param.Name == "template" {
			templateRequired = param.Required
		}
	}

	assert.ElementsMatch(t, []string{"template"}, paramNames)
	assert.True(t, templateRequired, "template parameter must be required")
}

func TestStart_HandleHook_EmitsWithConfiguredPayload(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{"message": "Hello, World!"}},
			map[string]any{"name": "Bye", "payload": map[string]any{"message": "Goodbye"}},
		},
	}

	result, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
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

func TestStart_HandleHook_UnknownTemplateListsAvailable(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{}},
			map[string]any{"name": "Bye", "payload": map[string]any{}},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
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

func TestStart_HandleHook_RejectsMissingTemplate(t *testing.T) {
	s := &Start{}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{},
		Configuration: map[string]any{},
		Events:        &contexts.EventContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template")
}

func TestStart_HandleHook_RejectsNoTemplatesConfigured(t *testing.T) {
	s := &Start{}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{"template": "Hello"},
		Configuration: map[string]any{},
		Events:        &contexts.EventContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates configured")
}

func TestStart_HandleHook_EmitsWithConfiguredParameters(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Hello",
				"payload": map[string]any{"message": "Hello, World!"},
				"parameters": []any{
					map[string]any{"name": "message", "type": "string", "defaultString": "Hello, World!"},
				},
			},
		},
	}

	result, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{"template": "Hello"},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello", result["template"])

	require.Len(t, events.Payloads, 1)
	payload, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, World!", payload["message"])
}

func TestStart_HandleHook_PrefersPayloadOverParameters(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Hello",
				"payload": map[string]any{"message": "from payload"},
				"parameters": []any{
					map[string]any{"name": "message", "type": "string", "defaultString": "from parameters"},
				},
			},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{"template": "Hello"},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	payload := events.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "from payload", payload["message"])
}

func TestStart_HandleHook_RejectsNilPayload(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Bad", "payload": nil},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{"template": "Bad"},
		Configuration: config,
		Events:        events,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no payload")
	assert.Empty(t, events.Payloads)
}

func TestStart_Configuration_SelectParameterRequiresOptions(t *testing.T) {
	s := &Start{}

	err := configuration.ValidateConfiguration(s.Configuration(), map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Parameterized",
				"payload": map[string]any{"provider": "x"},
				"parameters": []any{
					map[string]any{
						"name": "provider",
						"type": "select",
					},
				},
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "options")
}

func TestStart_Configuration_NonSelectParameterDoesNotRequireOptions(t *testing.T) {
	s := &Start{}

	err := configuration.ValidateConfiguration(s.Configuration(), map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Parameterized",
				"payload": map[string]any{"message": "hi"},
				"parameters": []any{
					map[string]any{
						"name": "message",
						"type": "string",
					},
				},
			},
		},
	})

	require.NoError(t, err)
}

func TestStart_Configuration_AcceptsTextParameter(t *testing.T) {
	s := &Start{}

	err := configuration.ValidateConfiguration(s.Configuration(), map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Prompted",
				"payload": map[string]any{"prompt": "hi"},
				"parameters": []any{
					map[string]any{
						"name":          "prompt",
						"type":          "text",
						"defaultString": "Write about\nsomething useful.",
					},
				},
			},
		},
	})

	require.NoError(t, err)
}

func TestStart_Configuration_TextParameterRejectsNonString(t *testing.T) {
	s := &Start{}

	err := configuration.ValidateConfiguration(s.Configuration(), map[string]any{
		"templates": []any{
			map[string]any{
				"name":    "Prompted",
				"payload": map[string]any{"prompt": "hi"},
				"parameters": []any{
					map[string]any{
						"name":          "prompt",
						"type":          "text",
						"defaultString": 42,
					},
				},
			},
		},
	})

	require.Error(t, err)
}

func TestStart_HandleHook_RejectsUnknownHook(t *testing.T) {
	s := &Start{}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:   "nope",
		Events: &contexts.EventContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
