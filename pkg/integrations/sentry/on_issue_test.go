package sentry

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}
	t.Run("valid project persists subscription and project metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "backend",
				"actions": []string{"created"},
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		metadata, ok := metadataCtx.Metadata.(OnIssueMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.AppSubscriptionID)
		require.NotNil(t, metadata.Project)
		assert.Equal(t, "backend", metadata.Project.Slug)
	})

	t.Run("invalid project does not create a subscription", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "missing",
				"actions": []string{"created"},
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.ErrorContains(t, err, `project "missing" was not found`)
		assert.Empty(t, integrationCtx.Subscriptions)
		assert.Nil(t, metadataCtx.Metadata)
	})

	t.Run("does not panic when the trigger has no integration", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		require.NotPanics(t, func() {
			err := trigger.Setup(core.TriggerContext{
				Configuration: map[string]any{"actions": []string{"created"}},
				Metadata:      metadata,
			})
			require.NoError(t, err)
		})
	})

	t.Run("requires an integration when a project is selected", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "production",
				"actions": []string{"created"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Equal(t, "Sentry integration is not connected", err.Error())
	})
}

func Test__OnIssue__OnIntegrationMessage(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	issue := map[string]any{
		"id":        "123",
		"title":     "Broken deploy",
		"permalink": "https://your-org.sentry.io/issues/123/",
		"project": map[string]any{
			"slug": "backend",
		},
	}
	message := WebhookMessage{
		Resource: "issue",
		Action:   "resolved",
		Data: map[string]any{
			"issue": issue,
		},
	}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: message,
		Configuration: map[string]any{
			"project": "backend",
			"actions": []string{"resolved"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)
	assert.Equal(t, "sentry.issue", eventCtx.Payloads[0].Type)

	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, IssueDescription(issue, nil), payload["description"])
	assert.Equal(t, issue, payload["data"].(map[string]any)["issue"])
}

func Test__OnIssue__OnIntegrationMessage__EmptyActionsEmitNothing(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "created",
			Data: map[string]any{
				"issue": map[string]any{"id": "123"},
			},
		},
		Configuration: map[string]any{
			"actions": []string{},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.Empty(t, eventCtx.Payloads)
}

func Test__OnIssue__OnIntegrationMessage__EnrichesDescriptionFromAPI(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{
				"id":"123",
				"title":"TypeError: boom",
				"permalink":"https://your-org.sentry.io/issues/123/",
				"count":"8",
				"status":"unresolved",
				"project":{"name":"Backend","slug":"backend"}
			}`),
			sentryMockResponse(http.StatusOK, `{
				"eventID":"evt-latest",
				"entries":[{"type":"exception","data":{"values":[{"type":"TypeError","value":"boom","stacktrace":{"frames":[{"filename":"app.go","function":"Handle","lineNo":22,"inApp":true}]}}]}}]
			}`),
		},
	}

	issue := map[string]any{
		"id":        "123",
		"title":     "TypeError: boom",
		"permalink": "https://your-org.sentry.io/issues/123/",
		"project":   map[string]any{"slug": "backend"},
	}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "created",
			Data:     map[string]any{"issue": issue},
		},
		Configuration: map[string]any{"actions": []string{"created"}},
		Events:        eventCtx,
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "user-token"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
		},
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	description, _ := payload["description"].(string)
	assert.Contains(t, description, "## Stack Trace")
	assert.Contains(t, description, "Handle (app.go:22) [in app]")
	assert.Contains(t, description, "**Count:** 8")
	assert.NotContains(t, description, "```json")
	require.Len(t, httpCtx.Requests, 2)
}

func Test__OnIssue__OnIntegrationMessage__KeepsWebhookWhenEnrichmentFails(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusInternalServerError, `{"detail":"error"}`),
			sentryMockResponse(http.StatusInternalServerError, `{"detail":"error"}`),
		},
	}

	issue := map[string]any{
		"id":        "123",
		"title":     "Broken deploy",
		"permalink": "https://your-org.sentry.io/issues/123/",
	}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "created",
			Data:     map[string]any{"issue": issue},
		},
		Configuration: map[string]any{"actions": []string{"created"}},
		Events:        eventCtx,
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "user-token"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
		},
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, IssueDescription(issue, nil), payload["description"])
}

func Test__OnIssue__OnIntegrationMessage__MissingIssue(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "created",
			Data:     map[string]any{},
		},
		Configuration: map[string]any{
			"actions": []string{"created"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)

	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Empty(t, payload["description"])
}

func Test__OnIssue__OnIntegrationMessage__UsesTopLevelWebhookTimestamp(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource:  "issue",
			Action:    "resolved",
			Timestamp: "2026-03-24T10:15:00Z",
			Data: map[string]any{
				"issue": map[string]any{
					"id":       "123",
					"title":    "Broken deploy",
					"lastSeen": "2026-03-20T09:00:00Z",
				},
			},
		},
		Configuration: map[string]any{
			"actions": []string{"resolved"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)

	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2026-03-24T10:15:00Z", payload["timestamp"])
}

func Test__OnIssue__OnIntegrationMessage__ArchivedAction(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "archived",
			Data: map[string]any{
				"issue": map[string]any{
					"id": "123",
				},
			},
		},
		Configuration: map[string]any{
			"actions": []string{"archived"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)
	assert.Equal(t, "sentry.issue", eventCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}
