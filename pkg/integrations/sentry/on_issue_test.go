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

	t.Run("project added after connection setup creates a subscription", func(t *testing.T) {
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
				"project": "new-project",
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
		assert.Equal(t, "new-project", metadata.Project.Slug)
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
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.example.com"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "your-org"}},
		},
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
	assert.Equal(t, "https://sentry.example.com/api/0/organizations/your-org/issues/123/", payload["issueApiUrl"])
	assert.Equal(t, issue, payload["data"].(map[string]any)["issue"])
}

func Test__OnIssue__OnIntegrationMessage__ProjectsList(t *testing.T) {
	issue := func(slug string) WebhookMessage {
		return WebhookMessage{
			Resource: "issue",
			Action:   "created",
			Data: map[string]any{
				"issue": map[string]any{
					"id":      "123",
					"project": map[string]any{"slug": slug},
				},
			},
		}
	}

	t.Run("emits only for a project in the list", func(t *testing.T) {
		trigger := &OnIssue{}
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: issue("growth"),
			Configuration: map[string]any{
				"project":  "",
				"projects": []any{"payments", "growth"},
				"actions":  []string{"created"},
			},
			Events: eventCtx,
			Logger: logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		require.Len(t, eventCtx.Payloads, 1)
	})

	t.Run("drops an issue from another project", func(t *testing.T) {
		trigger := &OnIssue{}
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: issue("billing"),
			Configuration: map[string]any{
				"project":  "payments",
				"projects": []any{"payments", "growth"},
				"actions":  []string{"created"},
			},
			Events: eventCtx,
			Logger: logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		assert.Empty(t, eventCtx.Payloads)
	})

	t.Run("an empty list still honors the single project", func(t *testing.T) {
		trigger := &OnIssue{}
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: issue("billing"),
			Configuration: map[string]any{
				"project":  "payments",
				"projects": []any{},
				"actions":  []string{"created"},
			},
			Events: eventCtx,
			Logger: logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		assert.Empty(t, eventCtx.Payloads)

		allowed := &contexts.EventContext{}
		err = trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: issue("payments"),
			Configuration: map[string]any{
				"project": "payments",
				"actions": []string{"created"},
			},
			Events: allowed,
			Logger: logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		require.Len(t, allowed.Payloads, 1)
	})
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

func Test__OnIssue__OnIntegrationMessage__OmittedActionsEmitEvents(t *testing.T) {
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
		Configuration: map[string]any{},
		Events:        eventCtx,
		Logger:        logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)
	assert.Equal(t, "sentry.issue", eventCtx.Payloads[0].Type)
}

func Test__OnIssue__OnIntegrationMessage__UsesWebhookWithoutSentryAPI(t *testing.T) {
	trigger := &OnIssue{}
	events := &contexts.EventContext{}
	httpCtx := &contexts.HTTPContext{}
	issue := map[string]any{
		"id": "123", "title": "Broken deploy",
		"permalink": "https://sentry.io/issues/123/",
	}
	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue", Action: "created", ReceiptID: "receipt-1",
			Data: map[string]any{"issue": issue},
		},
		Configuration: map[string]any{"actions": []string{"created"}},
		Events:        events,
		HTTP:          httpCtx,
		Integration:   &contexts.IntegrationContext{},
		Logger:        logrus.NewEntry(logrus.New()),
	})
	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	payload := events.Payloads[0].Data.(map[string]any)
	assert.Equal(t, IssueDescription(issue, nil), payload["description"])
	assert.Equal(t, "receipt-1", payload[SuperplaneReceiptField])
	assert.Empty(t, httpCtx.Requests)
}

func TestOnIssueUsesHostedSentryAPIURL(t *testing.T) {
	events := &contexts.EventContext{}
	err := (&OnIssue{}).OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue", Action: "created",
			Data: map[string]any{"issue": map[string]any{"id": "123"}},
		},
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{HostedApp: true, Organization: &OrganizationSummary{Slug: "example"}},
		},
		Events: events,
	})

	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	payload := events.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", payload["issueApiUrl"])
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
