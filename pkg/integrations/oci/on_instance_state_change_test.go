package oci

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test__OnInstanceStateChange__Setup_UsesWebhookPathIDForRuleName(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, `{"id":"ocid1.eventsrule.oc1.test"}`),
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{"compartmentId": testCompartmentID},
		HTTP:          httpCtx,
		Integration:   ociIntegrationContext(),
		Metadata:      &contexts.MetadataContext{},
		Webhook:       fixedNodeWebhookContext{url: "https://example.com/api/v1/webhooks/webhook-id?token=abc#frag"},
		Logger:        ociLogger(),
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/20181201/rules")
	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"displayName":"superplane-instance-state-change-webhook-id"`)
	assert.NotContains(t, string(body), "token=abc")
}

func Test__OnInstanceStateChange__HandleWebhook_IgnoresUnknownEventTypes(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	events := &contexts.EventContext{}

	status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.launchinstance.end", testCompartmentID),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartmentId": testCompartmentID,
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
		Body:    instanceStateChangeEventBody(t, "com.oraclecloud.computeapi.stopinstance.end", "other-compartment"),
		Events:  events,
		Headers: http.Header{},
		Configuration: map[string]any{
			"compartmentId": testCompartmentID,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, events.Count())
}

func Test__OnInstanceStateChange__HandleWebhook_EmitsValidEventTypes(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	validEventTypes := []string{
		"com.oraclecloud.computeapi.startinstance.end",
		"com.oraclecloud.computeapi.stopinstance.end",
		"com.oraclecloud.computeapi.terminateinstance.end",
		"com.oraclecloud.computeapi.resetinstance.end",
		"com.oraclecloud.computeapi.softstopinstance.end",
		"com.oraclecloud.computeapi.softresetinstance.end",
	}

	for _, eventType := range validEventTypes {
		t.Run(eventType, func(t *testing.T) {
			events := &contexts.EventContext{}

			status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
				Body:    instanceStateChangeEventBody(t, eventType, testCompartmentID),
				Events:  events,
				Headers: http.Header{},
				Configuration: map[string]any{
					"compartmentId": testCompartmentID,
				},
			})

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
			require.Len(t, events.Payloads, 1)
			assert.Equal(t, OnInstanceStateChangePayloadType, events.Payloads[0].Type)
		})
	}
}

func instanceStateChangeEventBody(t *testing.T, eventType, compartmentID string) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"eventType": eventType,
		"eventTime": "2026-04-22T20:34:54Z",
		"data": map[string]any{
			"resourceId":         testInstanceID,
			"resourceName":       "test-instance",
			"compartmentId":      compartmentID,
			"compartmentName":    "root",
			"availabilityDomain": "XXXX:eu-frankfurt-1-AD-1",
			"additionalDetails": map[string]any{
				"shape": "VM.Standard.E2.1.Micro",
			},
		},
	})
	require.NoError(t, err)
	return body
}

type fixedNodeWebhookContext struct {
	url string
}

func (w fixedNodeWebhookContext) Setup() (string, error) {
	return w.url, nil
}

func (w fixedNodeWebhookContext) GetSecret() ([]byte, error) {
	return nil, nil
}

func (w fixedNodeWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return nil, nil, nil
}

func (w fixedNodeWebhookContext) SetSecret(secret []byte) error {
	return nil
}

func (w fixedNodeWebhookContext) GetBaseURL() string {
	return "http://localhost:3000/api/v1"
}
