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

	t.Run("always matches, even with different events - Jira allows only one registered URL per connection", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{issueEventCreated}},
			WebhookConfiguration{Events: []string{commentEventCreated}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})
}

func Test__WebhookHandler__Merge(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("widens events when the requested config adds a new one", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted}}
		requested := WebhookConfiguration{Events: []string{commentEventCreated}}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{
			Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted, commentEventCreated},
		}, merged)
	})

	t.Run("reports no change when the requested events are already covered", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{issueEventCreated, commentEventCreated}}
		requested := WebhookConfiguration{Events: []string{commentEventCreated}}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})

	t.Run("treats an empty stored config as the legacy issue-event baseline, not as empty", func(t *testing.T) {
		current := WebhookConfiguration{}
		requested := WebhookConfiguration{Events: []string{commentEventCreated}}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{
			Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted, commentEventCreated},
		}, merged)
	})

	// Regression test: re-saving a jira.onIssue trigger on a legacy webhook must not force a
	// destructive re-provision every time - only genuinely new events should (BUGBOT_BUG_ID
	// 4df366b5).
	t.Run("reports no change when a request against a legacy row is already covered by the baseline", func(t *testing.T) {
		current := WebhookConfiguration{}
		requested := WebhookConfiguration{Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted}}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
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
			Webhook: &contexts.WebhookContext{
				URL: "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{
					Events: []string{
						issueEventCreated, issueEventUpdated, issueEventDeleted,
						commentEventCreated, commentEventUpdated, commentEventDeleted,
					},
				},
			},
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
		assert.Contains(t, string(body), `"comment_created"`)
		assert.Contains(t, string(body), `"comment_updated"`)
		assert.Contains(t, string(body), `"comment_deleted"`)
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

	// Regression test: re-running Setup on a webhook that already has a Jira registration (e.g. a
	// jira.onIssueComment trigger widening an integration's events via Merge) must replace that
	// registration instead of leaving it orphaned - Jira allows only one per connection.
	t.Run("re-provisioning replaces the previous Jira registration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":2000}]`))},
			},
		}
		integration := newAuthorizedIntegration()
		previousID := int64(1000)

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Metadata:      WebhookMetadata{WebhookID: &previousID},
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated, commentEventCreated}},
			},
		})
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, int64(2000), *webhookMetadata.WebhookID)

		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), "1000")
		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	})

	// Regression test: deleting a working shared webhook before its replacement exists silences
	// every trigger sharing it if the create then fails - retry the create tightly so a transient
	// failure recovers in seconds instead of waiting for the provisioner's own retry cadence
	// (BUGBOT_BUG_ID 2fb96a18).
	t.Run("retries the create after a transient failure following a delete", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["timeout"]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":3000}]`))},
			},
		}
		integration := newAuthorizedIntegration()
		previousID := int64(1000)

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Metadata:      WebhookMetadata{WebhookID: &previousID},
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated}},
			},
		})
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, int64(3000), *webhookMetadata.WebhookID)
		require.Len(t, httpCtx.Requests, 3)
	})

	t.Run("surfaces the error once retries are exhausted after a delete", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["down"]}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["down"]}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["down"]}`))},
			},
		}
		integration := newAuthorizedIntegration()
		previousID := int64(1000)

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Metadata:      WebhookMetadata{WebhookID: &previousID},
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated}},
			},
		})
		require.ErrorContains(t, err, "failed to create Jira webhook")
		require.Len(t, httpCtx.Requests, 4)
	})

	t.Run("falls back to the legacy issue-event baseline when Events is empty", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
			},
		}
		integration := newAuthorizedIntegration()

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{URL: "https://sp.test/webhooks/w1"},
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), `"jira:issue_created"`)
		assert.Contains(t, string(body), `"jira:issue_updated"`)
		assert.Contains(t, string(body), `"jira:issue_deleted"`)
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
