package planelet

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__ExecuteActionUsesV2Endpoint(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"serverUrl": "https://planelet.example/",
			"authToken": "test-token",
		},
	}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"success":true,"data":{"ok":true}}`),
		},
	}

	client, err := NewClientWithHTTP(integration, httpCtx)
	require.NoError(t, err)

	result, err := client.ExecuteAction("folder/create-page", map[string]any{"title": "Hello"}, map[string]any{"source": "test"})
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Equal(t, true, result.Data["ok"])

	require.Len(t, httpCtx.Requests, 1)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://planelet.example/actions/folder%2Fcreate-page/execute", req.URL.String())
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

	var body ExecuteRequest
	require.NoError(t, decodeRequestBody(req, &body))
	assert.Equal(t, "Hello", body.Parameters["title"])
	assert.Equal(t, "test", body.Input.(map[string]any)["source"])
}

func Test__Planelet__ListResourcesReturnsActionsAndTriggers(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"serverUrl": "https://planelet.example"},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, planeletManifestJSON()),
			jsonResponse(http.StatusOK, planeletManifestJSON()),
		},
	}

	resources, err := (&Planelet{}).ListResources("action", core.ListResourcesContext{
		Integration: integration,
		HTTP:        httpCtx,
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, core.IntegrationResource{Type: "action", ID: "create-page", Name: "Create Page"}, resources[0])

	resources, err = (&Planelet{}).ListResources("trigger", core.ListResourcesContext{
		Integration: integration,
		HTTP:        httpCtx,
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, core.IntegrationResource{Type: "trigger", ID: "database-created", Name: "Database Created"}, resources[0])
}

func Test__PlaneletComponents__UseManifestParameters(t *testing.T) {
	setCachedManifest(&Manifest{
		Actions: []ActionManifest{
			{
				ID:    "create-page",
				Label: "Create Page",
				Parameters: []ParameterManifest{
					{ID: "databaseId", Label: "Database ID", Type: "string", Required: true},
				},
			},
		},
		Triggers: []TriggerManifest{
			{
				ID:    "database-created",
				Label: "Database Created",
				Parameters: []ParameterManifest{
					{ID: "workspaceId", Label: "Workspace ID", Type: "string", Required: true},
				},
			},
		},
	})
	t.Cleanup(func() { setCachedManifest(nil) })

	actionFields := (&RunAction{}).Configuration()
	require.Len(t, actionFields, 2)
	assert.Equal(t, "actionId", actionFields[0].Name)
	assert.Equal(t, "param_create-page_databaseId", actionFields[1].Name)
	assert.Equal(t, "actionId", actionFields[1].VisibilityConditions[0].Field)
	assert.Equal(t, []string{"create-page"}, actionFields[1].VisibilityConditions[0].Values)

	triggerFields := (&WebhookTrigger{}).Configuration()
	require.Len(t, triggerFields, 2)
	assert.Equal(t, "triggerId", triggerFields[0].Name)
	assert.Equal(t, "param_database-created_workspaceId", triggerFields[1].Name)
	assert.Equal(t, "triggerId", triggerFields[1].VisibilityConditions[0].Field)
	assert.Equal(t, []string{"database-created"}, triggerFields[1].VisibilityConditions[0].Values)
}

func Test__WebhookTrigger__SetupCallsPlaneletServer(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"serverUrl": "https://planelet.example",
			"authToken": "test-token",
		},
	}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, planeletManifestJSON()),
			jsonResponse(http.StatusOK, `{"success":true,"metadata":{"providerWebhookId":"wh_123"}}`),
		},
	}

	metadataCtx := &contexts.MetadataContext{}
	err := (&WebhookTrigger{}).Setup(core.TriggerContext{
		Configuration: map[string]any{
			"triggerId":                                 "database-created",
			"param_database-created_workspaceId":        "ws_123",
			"param_some-other-trigger_ignoredParameter": "ignored",
		},
		HTTP:        httpCtx,
		Integration: integration,
		Metadata:    metadataCtx,
		Webhook:     &contexts.NodeWebhookContext{Secret: "superplane-secret"},
	})
	require.NoError(t, err)

	require.Len(t, httpCtx.Requests, 2)
	setupReq := httpCtx.Requests[1]
	assert.Equal(t, http.MethodPost, setupReq.Method)
	assert.Equal(t, "https://planelet.example/triggers/database-created/setup", setupReq.URL.String())
	assert.Equal(t, "Bearer test-token", setupReq.Header.Get("Authorization"))

	var body SetupTriggerRequest
	require.NoError(t, decodeRequestBody(setupReq, &body))
	assert.Equal(t, "ws_123", body.Parameters["workspaceId"])
	assert.NotEmpty(t, body.Webhook.URL)
	assert.Equal(t, "superplane-secret", body.Webhook.Secret)

	metadata, ok := metadataCtx.Metadata.(WebhookTriggerMetadata)
	require.True(t, ok)
	assert.Equal(t, "database-created", metadata.TriggerID)
	assert.Equal(t, "wh_123", metadata.PlaneletMetadata["providerWebhookId"])
	assert.Equal(t, "ws_123", metadata.Parameters["workspaceId"])
}

func Test__WebhookTrigger__HandleWebhookNormalizesAndEmitsEvent(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"serverUrl": "https://planelet.example"},
	}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{
				"success": true,
				"emit": true,
				"eventType": "notion.database.created",
				"payload": {"database":{"id":"db_123"}},
				"response": {
					"status": 202,
					"headers": {"Content-Type":"text/plain","X-Planelet":"ok"},
					"body": "accepted"
				}
			}`),
		},
	}

	events := &contexts.EventContext{}
	status, response, err := (&WebhookTrigger{}).HandleWebhook(core.WebhookRequestContext{
		Body:    []byte(`{"database":{"id":"db_123"}}`),
		Method:  http.MethodPost,
		Headers: http.Header{"X-Notion-Signature": []string{"sig"}},
		Query:   map[string][]string{"challenge": {"abc"}},
		Configuration: map[string]any{
			"triggerId":                          "database-created",
			"param_database-created_workspaceId": "ws_123",
		},
		Metadata: &contexts.MetadataContext{
			Metadata: WebhookTriggerMetadata{
				TriggerID:        "database-created",
				PlaneletMetadata: map[string]any{"providerWebhookId": "wh_123"},
			},
		},
		HTTP:        httpCtx,
		Integration: integration,
		Events:      events,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, status)
	require.NotNil(t, response)
	assert.Equal(t, "accepted", string(response.Body))
	assert.Equal(t, "text/plain", response.ContentType)
	assert.Equal(t, "ok", response.Headers["X-Planelet"])

	require.Len(t, events.Payloads, 1)
	assert.Equal(t, "notion.database.created", events.Payloads[0].Type)
	assert.Equal(t, "db_123", events.Payloads[0].Data.(map[string]any)["database"].(map[string]any)["id"])

	require.Len(t, httpCtx.Requests, 1)
	var forwarded HandleTriggerWebhookRequest
	require.NoError(t, decodeRequestBody(httpCtx.Requests[0], &forwarded))
	assert.Equal(t, "ws_123", forwarded.Parameters["workspaceId"])
	assert.Equal(t, "wh_123", forwarded.Metadata["providerWebhookId"])
	assert.Equal(t, http.MethodPost, forwarded.Request.Method)
	assert.Equal(t, []string{"sig"}, forwarded.Request.Headers["X-Notion-Signature"])
	assert.Equal(t, []string{"abc"}, forwarded.Request.Query["challenge"])
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(`{"database":{"id":"db_123"}}`)), forwarded.Request.RawBodyBase64)
}

func Test__WebhookTrigger__CleanupCallsPlaneletServer(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"serverUrl": "https://planelet.example"},
	}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"success":true}`),
		},
	}

	err := (&WebhookTrigger{}).Cleanup(core.TriggerContext{
		Configuration: map[string]any{"triggerId": "database-created"},
		Metadata: &contexts.MetadataContext{
			Metadata: WebhookTriggerMetadata{
				TriggerID:        "database-created",
				Parameters:       map[string]any{"workspaceId": "ws_123"},
				PlaneletMetadata: map[string]any{"providerWebhookId": "wh_123"},
			},
		},
		HTTP:        httpCtx,
		Integration: integration,
	})
	require.NoError(t, err)

	require.Len(t, httpCtx.Requests, 1)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://planelet.example/triggers/database-created/cleanup", req.URL.String())

	var body CleanupTriggerRequest
	require.NoError(t, decodeRequestBody(req, &body))
	assert.Equal(t, "ws_123", body.Parameters["workspaceId"])
	assert.Equal(t, "wh_123", body.Metadata["providerWebhookId"])
}

func planeletManifestJSON() string {
	return `{
		"id": "notion",
		"label": "Notion",
		"iconUrl": "https://example.com/notion.svg",
		"actions": [
			{
				"id": "create-page",
				"label": "Create Page",
				"parameters": [
					{"id": "databaseId", "label": "Database ID", "type": "string", "required": true}
				]
			}
		],
		"triggers": [
			{
				"id": "database-created",
				"label": "Database Created",
				"parameters": [
					{"id": "workspaceId", "label": "Workspace ID", "type": "string", "required": true}
				]
			}
		]
	}`
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func decodeRequestBody(request *http.Request, v any) error {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, v)
}
