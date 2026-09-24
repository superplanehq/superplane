package jira

import (
	"encoding/json"
	"fmt"
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

	t.Run("issue/comment webhooks always match, even with different events - Jira allows only one registered URL per connection", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{issueEventCreated}},
			WebhookConfiguration{Events: []string{commentEventCreated}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	// Regression test: publishing an unchanged jira.onAlert trigger must not tear down and
	// recreate its JSM Ops integration on every commit - alert webhooks with the same team
	// dedup onto one registration, just like the shared issue/comment webhook (BUGBOT_BUG_ID
	// b743aa09).
	t.Run("alert webhooks with the same team match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("alert webhooks for different teams never match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"},
			WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-2"},
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

	t.Run("widens projects when the requested config adds a new one", func(t *testing.T) {
		current := WebhookConfiguration{
			Events:   []string{issueEventCreated},
			Projects: []string{"ENG"},
		}
		requested := WebhookConfiguration{
			Events:   []string{issueEventCreated},
			Projects: []string{"OPS"},
		}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{
			Events:   []string{issueEventCreated},
			Projects: []string{"ENG", "OPS"},
		}, merged)
	})

	t.Run("reports no change when the requested projects are already covered", func(t *testing.T) {
		current := WebhookConfiguration{
			Events:   []string{issueEventCreated},
			Projects: []string{"ENG", "OPS"},
		}
		requested := WebhookConfiguration{
			Events:   []string{issueEventCreated},
			Projects: []string{"ENG"},
		}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})

	t.Run("adds the requested project to a legacy row that stored none", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted}}
		requested := WebhookConfiguration{
			Events:   []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
			Projects: []string{"ENG"},
		}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{
			Events:   []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
			Projects: []string{"ENG"},
		}, merged)
	})
}

func Test__issueWebhookJQLFilter(t *testing.T) {
	t.Run("a single project uses equality", func(t *testing.T) {
		jql, err := issueWebhookJQLFilter([]string{"ENG"})
		require.NoError(t, err)
		assert.Equal(t, `project = "ENG"`, jql)
	})

	t.Run("several projects use IN", func(t *testing.T) {
		jql, err := issueWebhookJQLFilter([]string{"ENG", "OPS"})
		require.NoError(t, err)
		assert.Equal(t, `project IN ("ENG","OPS")`, jql)
	})

	t.Run("blank keys are ignored", func(t *testing.T) {
		jql, err := issueWebhookJQLFilter([]string{"", " ENG ", "OPS"})
		require.NoError(t, err)
		assert.Equal(t, `project IN ("ENG","OPS")`, jql)
	})

	t.Run("no project is an error", func(t *testing.T) {
		_, err := issueWebhookJQLFilter(nil)
		require.ErrorContains(t, err, "at least one project")

		_, err = issueWebhookJQLFilter([]string{"", "  "})
		require.ErrorContains(t, err, "at least one project")
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
					Projects: []string{"ENG"},
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
		// Dynamic webhook JQL only accepts project with =, !=, IN, or NOT IN.
		// project != EMPTY can register and still match nothing.
		assert.Contains(t, string(body), `"jqlFilter":"project = \"ENG\""`)

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

	t.Run("registers project IN when several triggers share the webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL: "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{
					Events:   []string{issueEventCreated},
					Projects: []string{"ENG", "OPS"},
				},
			},
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), `"jqlFilter":"project IN (\"ENG\",\"OPS\")"`)
	})

	t.Run("refuses to register an issue webhook without a project", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated}},
			},
		})
		require.ErrorContains(t, err, "at least one project")
		assert.Empty(t, httpCtx.Requests)
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
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated, commentEventCreated}, Projects: []string{"ENG"}},
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
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated}, Projects: []string{"ENG"}},
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
				Configuration: WebhookConfiguration{Events: []string{issueEventCreated}, Projects: []string{"ENG"}},
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
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Projects: []string{"ENG"}},
			},
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
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Projects: []string{"ENG"}},
			},
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
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/webhooks/w1",
				Configuration: WebhookConfiguration{Projects: []string{"ENG"}},
			},
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

const (
	testRequestedWebhookURL = "https://app.superplane.com/api/v1/webhooks/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testBlockerWebhookURL   = "https://app.superplane.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6"
	testBlockerWebhookID    = "ed13e750-bd47-4ae4-9400-4d75ef42eab6"
)

func urlConflictResponse(blockerURL string) *http.Response {
	body := fmt.Sprintf(
		`{"webhookRegistrationResult":[{"errors":["Only a single URL per user is allowed to be registered via REST API. The currently used URL: %s"]}]}`,
		blockerURL,
	)
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func Test__WebhookHandler__Setup__RecoversStaleURLConflict(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("deletes the matching stale registration and retries create", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse(testBlockerWebhookURL),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": true,
					"values": [
						{"id":1000,"url":"` + testBlockerWebhookURL + `"},
						{"id":1001,"url":"https://app.superplane.com/api/v1/webhooks/cccccccc-cccc-4ccc-8ccc-cccccccccccc"}
					]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":2000}]`))},
			},
		}
		integration := newAuthorizedIntegration()

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, int64(2000), *webhookMetadata.WebhookID)

		require.Len(t, httpCtx.Requests, 4)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[2].Method)
		body, _ := io.ReadAll(httpCtx.Requests[2].Body)
		assert.Contains(t, string(body), "1000")
		assert.NotContains(t, string(body), "1001")
		assert.Equal(t, http.MethodPost, httpCtx.Requests[3].Method)
	})

	t.Run("paginates the remote list before deleting the blocker", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse(testBlockerWebhookURL),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": false,
					"values": [{"id":999,"url":"https://app.superplane.com/api/v1/webhooks/cccccccc-cccc-4ccc-8ccc-cccccccccccc"}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": true,
					"values": [{"id":1000,"url":"` + testBlockerWebhookURL + `"}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":2000}]`))},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 5)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[3].Method)
	})

	t.Run("clears a mirrored integration webhook id that was deleted", func(t *testing.T) {
		staleID := int64(1000)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse(testBlockerWebhookURL),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": true,
					"values": [{"id":1000,"url":"` + testBlockerWebhookURL + `"}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":2000}]`))},
			},
		}
		integration := newAuthorizedIntegrationWithMetadata(Metadata{
			CloudID:                     testCloudID,
			SiteURL:                     testSiteURL,
			IssueWebhookScopesRequested: true,
			WebhookID:                   &staleID,
		})

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integration,
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.NoError(t, err)

		storedMetadata := Metadata{}
		require.NoError(t, mapstructure.Decode(integration.Metadata, &storedMetadata))
		require.NotNil(t, storedMetadata.WebhookID)
		assert.Equal(t, int64(2000), *storedMetadata.WebhookID)
	})

	t.Run("refuses takeover when the blocking webhook still has active consumers", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse(testBlockerWebhookURL),
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:             testRequestedWebhookURL,
				Configuration:   WebhookConfiguration{Projects: []string{"WRK3"}},
				ActiveCallbacks: map[string]bool{testBlockerWebhookID: true},
			},
		})
		require.ErrorContains(t, err, "still has active consumers")
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("refuses a blocking URL on a different origin", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse("https://other.example.com/api/v1/webhooks/" + testBlockerWebhookID),
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.ErrorContains(t, err, "not a SuperPlane callback on this origin")
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("surfaces an empty or mismatched Jira listing", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				urlConflictResponse(testBlockerWebhookURL),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": true,
					"values": [{"id":1001,"url":"https://app.superplane.com/api/v1/webhooks/cccccccc-cccc-4ccc-8ccc-cccccccccccc"}]
				}`))},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.ErrorContains(t, err, "no registered Jira webhook matches")
		require.Len(t, httpCtx.Requests, 2)
	})

	t.Run("recovers a URL conflict after replacing a previous registration", func(t *testing.T) {
		previousID := int64(500)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				urlConflictResponse(testBlockerWebhookURL),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"isLast": true,
					"values": [{"id":1000,"url":"` + testBlockerWebhookURL + `"}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":2000}]`))},
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           testRequestedWebhookURL,
				Metadata:      WebhookMetadata{WebhookID: &previousID},
				Configuration: WebhookConfiguration{Projects: []string{"WRK3"}},
			},
		})
		require.NoError(t, err)
		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, int64(2000), *webhookMetadata.WebhookID)
		require.Len(t, httpCtx.Requests, 5)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[3].Method)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[4].Method)
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

func TestIssueURL(t *testing.T) {
	assert.Equal(t, "https://acme.atlassian.net/browse/ENG-42", IssueURL("https://acme.atlassian.net", "ENG-42"))
	assert.Equal(t, "https://acme.atlassian.net/browse/ENG-42", IssueURL("https://acme.atlassian.net/", "ENG-42"))
	assert.Empty(t, IssueURL("", "ENG-42"))
	assert.Empty(t, IssueURL("https://acme.atlassian.net", ""))
}

func TestWebhookIssueEventCarriesBrowseURL(t *testing.T) {
	issue := &Issue{
		Key:  "ENG-42",
		Self: "https://api.atlassian.com/ex/jira/cloud-id/rest/api/3/issue/10001",
	}

	event := NewIssueEvent("created", issue, nil, nil, "https://acme.atlassian.net")
	assert.Equal(t, "https://acme.atlassian.net/browse/ENG-42", event.URL)
}
