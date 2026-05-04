package oci

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnInstanceStateChange__HandleWebhook_ConfirmsSubscription(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, `{}`),
		},
	}
	headers := http.Header{}
	headers.Set("X-OCI-NS-ConfirmationURL", "https://notification.eu-frankfurt-1.oraclecloud.com/confirm")

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    []byte(`not-json`),
		HTTP:    httpCtx,
		Events:  &contexts.EventContext{},
		Headers: headers,
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://notification.eu-frankfurt-1.oraclecloud.com/confirm", httpCtx.Requests[0].URL.String())
}

func Test__OnInstanceStateChange__Setup_RequestsWebhookWithoutEventsRule(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	httpCtx := &contexts.HTTPContext{}
	integration := ociIntegrationContext()

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{"compartment": testCompartmentID},
		HTTP:          httpCtx,
		Integration:   integration,
		Metadata:      &contexts.MetadataContext{},
		Logger:        ociLogger(),
	})

	require.NoError(t, err)
	assert.Empty(t, httpCtx.Requests, "instance state events use the integration-level Events rule from Sync, not a per-trigger rule")
	require.Len(t, integration.WebhookRequests, 1)
	cfg, ok := integration.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok, "expected WebhookConfiguration, got %T", integration.WebhookRequests[0])
	assert.Equal(t, testCompartmentID, cfg.CompartmentID)
	assert.Equal(t, "ocid1.onstopic.oc1.eu-frankfurt-1.testtopic", cfg.TopicID)
}

func Test__OnInstanceStateChange__Setup_RejectsUnsupportedStateChanges(t *testing.T) {
	trigger := &OnInstanceStateChange{}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{
			"compartment":  testCompartmentID,
			"stateChanges": []string{"hibernate"},
		},
		Integration: ociIntegrationContext(),
		Metadata:    &contexts.MetadataContext{},
		Logger:      ociLogger(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported state change: hibernate")
}

func Test__OnInstanceStateChange__Configuration(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	fields := trigger.Configuration()

	byName := map[string]configuration.Field{}
	for _, field := range fields {
		byName[field.Name] = field
	}

	stateChanges, ok := byName["stateChanges"]
	require.True(t, ok, "stateChanges field should exist")
	assert.Equal(t, configuration.FieldTypeMultiSelect, stateChanges.Type)
	assert.True(t, stateChanges.Required)
	assert.Equal(t, []string{
		ociInstanceStateChangeStart,
		ociInstanceStateChangeStop,
		ociInstanceStateChangeReset,
		ociInstanceStateChangeSoftStop,
		ociInstanceStateChangeSoftReset,
		ociInstanceStateChangeTerminate,
	}, stateChanges.Default)
	require.NotNil(t, stateChanges.TypeOptions)
	require.NotNil(t, stateChanges.TypeOptions.MultiSelect)
	assert.False(t, stateChanges.TypeOptions.MultiSelect.UseCheckboxes)
}

func Test__OnInstanceStateChange__HandleWebhook_IgnoresUnknownEventTypes(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.launchinstance.end", testCompartmentID, ""),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment": testCompartmentID,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_IgnoresUnknownInstanceActionTypes(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.instanceaction.end", testCompartmentID, "senddiagnosticinterrupt"),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment": testCompartmentID,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_FiltersUnselectedStateChanges(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.instanceaction.end", testCompartmentID, "stop"),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment":  testCompartmentID,
			"stateChanges": []string{"start"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_EmitsSelectedStateChanges(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.instanceaction.end", testCompartmentID, "softstop"),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment":  testCompartmentID,
			"stateChanges": []string{"softstop"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.Len(t, events.Payloads, 1)
	assert.Equal(t, OnInstanceStateChangePayloadType, events.Payloads[0].Type)
}

func Test__OnInstanceStateChange__HandleWebhook_FiltersTerminate(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.terminateinstance.end", testCompartmentID, ""),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment":  testCompartmentID,
			"stateChanges": []string{"stop"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_FiltersDifferentCompartment(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.instanceaction.end", "other-compartment", "stop"),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartment": testCompartmentID,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_EmitsValidEventTypes(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	validEvents := []struct {
		name       string
		eventType  string
		actionType string
	}{
		{name: "start", eventType: "com.oraclecloud.computeapi.instanceaction.end", actionType: "start"},
		{name: "stop", eventType: "com.oraclecloud.computeapi.instanceaction.end", actionType: "stop"},
		{name: "reset", eventType: "com.oraclecloud.computeapi.instanceaction.end", actionType: "reset"},
		{name: "softstop", eventType: "com.oraclecloud.computeapi.instanceaction.end", actionType: "softstop"},
		{name: "softreset", eventType: "com.oraclecloud.computeapi.instanceaction.end", actionType: "softreset"},
		{name: "terminate", eventType: "com.oraclecloud.computeapi.terminateinstance.end"},
	}

	for _, event := range validEvents {
		t.Run(event.name, func(t *testing.T) {
			events := &contexts.EventContext{}

			status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
				Body:    instanceStateChangeEventBody(t, event.eventType, testCompartmentID, event.actionType),
				Events:  events,
				Headers: http.Header{},
				Configuration: map[string]any{
					"compartment": testCompartmentID,
				},
			})

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
			require.Len(t, events.Payloads, 1)
			assert.Equal(t, OnInstanceStateChangePayloadType, events.Payloads[0].Type)
		})
	}
}

func instanceStateChangeEventBody(t *testing.T, eventType, compartmentID, actionType string) []byte {
	t.Helper()

	additionalDetails := map[string]any{
		"shape": "VM.Standard.E2.1.Micro",
	}
	if actionType != "" {
		additionalDetails["instanceActionType"] = actionType
	}

	body, err := json.Marshal(map[string]any{
		"eventType": eventType,
		"eventTime": "2026-04-22T20:34:54Z",
		"data": map[string]any{
			"resourceId":         testInstanceID,
			"resourceName":       "test-instance",
			"compartmentId":      compartmentID,
			"compartmentName":    "root",
			"availabilityDomain": "XXXX:eu-frankfurt-1-AD-1",
			"additionalDetails":  additionalDetails,
		},
	})
	require.NoError(t, err)
	return body
}
