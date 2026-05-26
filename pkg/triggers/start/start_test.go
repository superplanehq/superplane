package manual

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	var paramsRequired bool
	for _, param := range hook.Parameters {
		paramNames = append(paramNames, param.Name)
		if param.Name == "template" {
			templateRequired = param.Required
		}
		if param.Name == "params" {
			paramsRequired = param.Required
		}
	}

	assert.ElementsMatch(t, []string{"template", "params"}, paramNames)
	assert.True(t, templateRequired, "template parameter must be required")
	assert.False(t, paramsRequired, "params parameter must be optional")
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

func TestStart_HandleHook_ParamsStaticLeafOverride(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{"name": "Hello", "payload": map[string]any{"message": "Hello, World!"}},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name: HookRun,
		Parameters: map[string]any{
			"template": "Hello",
			"params":   map[string]any{"message": "Override"},
		},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	payload := events.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "Override", payload["message"])
}

func TestStart_HandleHook_ParamsSubstitutesParamLeaves(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{
				"name": "Deploy",
				"payload": map[string]any{
					"body": map[string]any{
						"name": "param(type:string, title:'Name', default:'machine-1', required:false)",
						"size": "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Size', required:true)",
					},
				},
			},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name: HookRun,
		Parameters: map[string]any{
			"template": "Deploy",
			"params": map[string]any{
				"body.size": "4 vCPU",
			},
		},
		Configuration: config,
		Events:        events,
	})

	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	payload := events.Payloads[0].Data.(map[string]any)
	body := payload["body"].(map[string]any)
	assert.Equal(t, "machine-1", body["name"])
	assert.Equal(t, "4 vCPU", body["size"])
}

func TestStart_HandleHook_RejectsMissingRequiredParam(t *testing.T) {
	s := &Start{}
	events := &contexts.EventContext{}

	config := map[string]any{
		"templates": []any{
			map[string]any{
				"name": "Deploy",
				"payload": map[string]any{
					"size": "param(type:select, values:'2 vCPU|4 vCPU', title:'Size', required:true)",
				},
			},
		},
	}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:          HookRun,
		Parameters:    map[string]any{"template": "Deploy"},
		Configuration: config,
		Events:        events,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "size")
	assert.Empty(t, events.Payloads)
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

func TestStart_HandleHook_RejectsUnknownHook(t *testing.T) {
	s := &Start{}

	_, err := s.HandleHook(core.TriggerHookContext{
		Name:   "nope",
		Events: &contexts.EventContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
