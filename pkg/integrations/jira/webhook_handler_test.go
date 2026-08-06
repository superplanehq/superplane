package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__WebhookHandler__CompareConfig(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("issue/comment webhooks always match - Jira allows only one registered URL per connection", func(t *testing.T) {
		equal, err := handler.CompareConfig(WebhookConfiguration{}, WebhookConfiguration{})
		require.NoError(t, err)
		assert.True(t, equal)
	})

	// Regression test: JSM Ops alert webhooks have no "one URL" restriction like Jira's dynamic
	// webhook API does - each jira.onAlert trigger must get its own dedicated registration rather
	// than sharing one, so these must never dedup even when the configs are otherwise identical.
	t.Run("alert webhooks never match, even with identical configuration", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("an alert webhook never matches an issue/comment webhook", func(t *testing.T) {
		equal, err := handler.CompareConfig(WebhookConfiguration{Kind: webhookKindAlert}, WebhookConfiguration{})
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__WebhookHandler__Setup(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("creates a webhook covering every project and event", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
			},
		}
		integration := newAuthorizedIntegration()

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{URL: "https://sp.test/webhooks/w1"},
		})
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		require.NotNil(t, webhookMetadata.WebhookID)
		assert.Equal(t, int64(1000), *webhookMetadata.WebhookID)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "/rest/api/3/webhook")

		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), `"jira:issue_created"`)
		assert.Contains(t, string(body), `"jira:issue_updated"`)
		assert.Contains(t, string(body), `"jira:issue_deleted"`)
		// Regression test: Atlassian rejects an empty jqlFilter outright ("Empty JQL search not
		// supported", confirmed live) even though the key must be present - this must be a real,
		// always-true clause instead.
		assert.Contains(t, string(body), `"jqlFilter":"project != EMPTY"`)

		// The id is mirrored onto the integration and a refresh is scheduled, since Atlassian
		// expires this webhook in 30 days otherwise.
		storedMetadata := Metadata{}
		require.NoError(t, mapstructure.Decode(integration.Metadata, &storedMetadata))
		require.NotNil(t, storedMetadata.WebhookID)
		assert.Equal(t, int64(1000), *storedMetadata.WebhookID)

		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, refreshWebhookHookName, integration.ActionRequests[0].ActionName)
		assert.Equal(t, webhookRefreshInterval, integration.ActionRequests[0].Interval)
	})

	t.Run("create failure is surfaced", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["bad request"]}`))},
			},
		}
		integration := newAuthorizedIntegration()

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{URL: "https://sp.test/webhooks/w1"},
		})
		require.ErrorContains(t, err, "failed to create Jira webhook")
		assert.Empty(t, integration.ActionRequests, "must not schedule a refresh when creation failed")
	})

	// Regression test: Jira allows only one URL per OAuth connection. If ScheduleActionCall fails
	// after CreateIssueWebhook succeeds, leaving the Jira registration behind makes retries fail
	// on that limit and leaves Cleanup unable to delete it (WebhookID never reaches the SuperPlane
	// webhook record). Setup must delete the registration on that path.
	t.Run("schedule failure rolls back the Jira registration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		integration := newAuthorizedIntegration()
		integration.ScheduleActionCallErr = assert.AnError

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{URL: "https://sp.test/webhooks/w1"},
		})
		require.ErrorContains(t, err, "failed to schedule webhook refresh")

		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[1].Method)
		body, _ := io.ReadAll(httpCtx.Requests[1].Body)
		assert.Contains(t, string(body), "1000")

		storedMetadata := Metadata{}
		require.NoError(t, mapstructure.Decode(integration.Metadata, &storedMetadata))
		assert.Nil(t, storedMetadata.WebhookID)
	})
}

func Test__WebhookHandler__Setup__AlertWebhook(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("creates a dedicated JSM Ops integration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"int-1"}`))},
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				ID:            "node-1",
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
			},
		})
		require.NoError(t, err)

		alertMetadata, ok := metadata.(*AlertWebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "int-1", alertMetadata.IntegrationID)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Contains(t, req.URL.String(), "/jsm/ops/api/"+testCloudID+"/v1/integrations")
		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), `"url":"https://sp.test/webhooks/w1"`)
		assert.Contains(t, string(body), `"teamId":"team-1"`)
		assert.Contains(t, string(body), "node-1")
	})

	t.Run("create failure is surfaced", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"message":"requires a Premium or Enterprise plan"}`))},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Kind: webhookKindAlert},
			},
		})
		require.ErrorContains(t, err, "Premium or Enterprise plan")
	})
}

func Test__WebhookHandler__Cleanup__AlertWebhook(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("deletes the registered integration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      AlertWebhookMetadata{IntegrationID: "int-1"},
				Configuration: WebhookConfiguration{Kind: webhookKindAlert},
			},
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "int-1")
	})

	t.Run("no-op when Setup never completed", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      AlertWebhookMetadata{},
				Configuration: WebhookConfiguration{Kind: webhookKindAlert},
			},
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__WebhookHandler__Cleanup(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("deletes the registered webhook and clears the mirrored id", func(t *testing.T) {
		webhookID := int64(1000)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		integration := newAuthorizedIntegrationWithMetadata(Metadata{WebhookID: &webhookID})

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: WebhookMetadata{WebhookID: &webhookID}},
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), "1000")

		storedMetadata := Metadata{}
		require.NoError(t, mapstructure.Decode(integration.Metadata, &storedMetadata))
		assert.Nil(t, storedMetadata.WebhookID)
	})

	t.Run("no-op when Setup never completed", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook:     &contexts.WebhookContext{Metadata: WebhookMetadata{}},
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("delete failure is surfaced", func(t *testing.T) {
		webhookID := int64(1000)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["not found"]}`))},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook:     &contexts.WebhookContext{Metadata: WebhookMetadata{WebhookID: &webhookID}},
		})
		require.ErrorContains(t, err, "not found")
	})
}

// Regression test: the platform persists webhook metadata by JSON-encoding it into a
// map[string]any (see models.Webhook.Metadata / WebhookContext.GetMetadata), which turns
// WebhookID into a float64 before mapstructure decodes it back - unlike the test doubles above,
// which store the struct directly. mapstructure (the version pinned in go.mod) does convert
// float64 into a *int64 field, so WebhookID survives the round trip; this pins that behavior so a
// future mapstructure upgrade can't silently break it.
func Test__WebhookMetadata__SurvivesJSONMetadataRoundTrip(t *testing.T) {
	webhookID := int64(1000)
	original := WebhookMetadata{WebhookID: &webhookID}

	serialized, err := json.Marshal(original)
	require.NoError(t, err)

	var asMap map[string]any
	require.NoError(t, json.Unmarshal(serialized, &asMap))

	// Confirms the premise: JSON decodes the number as float64, not int64.
	_, isFloat := asMap["webhookId"].(float64)
	require.True(t, isFloat)

	var restored WebhookMetadata
	require.NoError(t, mapstructure.Decode(asMap, &restored))

	require.NotNil(t, restored.WebhookID)
	assert.Equal(t, webhookID, *restored.WebhookID)
}
