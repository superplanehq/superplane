package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("valid setup requests shared webhook", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				AuthType: AuthTypeOAuth,
				CloudID:  "cloud-123",
				Projects: []Project{
					{ID: "10000", Key: "SP", Name: "SuperPlane"},
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project": "SP",
				"actions": []string{IssueActionCreated},
			},
		})

		require.NoError(t, err)
		require.Len(t, appCtx.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{CloudID: "cloud-123"}, appCtx.WebhookRequests[0])

		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, "SP", nodeMetadata.Project.Key)
	})

	t.Run("invalid action -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"actions": []string{"renamed"},
			},
		})

		require.ErrorContains(t, err, "unsupported issue action")
	})

	t.Run("requires completed oauth flow", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "not connected yet")
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("emits matching issue event", func(t *testing.T) {
		events := &contexts.EventContext{}
		status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       signedJiraHeaders(t, "secret"),
			Body:          []byte(issueWebhookPayload(jiraWebhookEventCreated, "SP")),
			Configuration: map[string]any{"project": "SP", "actions": []string{IssueActionCreated}},
			Events:        events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"clientSecret": "secret",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "jira.issue.created", events.Payloads[0].Type)
	})

	t.Run("filters unmatched project", func(t *testing.T) {
		events := &contexts.EventContext{}
		status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       signedJiraHeaders(t, "secret"),
			Body:          []byte(issueWebhookPayload(jiraWebhookEventUpdated, "OTHER")),
			Configuration: map[string]any{"project": "SP"},
			Events:        events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"clientSecret": "secret",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Empty(t, events.Payloads)
	})

	t.Run("accepts JWT prefix", func(t *testing.T) {
		events := &contexts.EventContext{}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": "jira",
			"exp": time.Now().Add(time.Hour).Unix(),
		})

		signed, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		headers := http.Header{"Authorization": []string{"JWT " + signed}}

		status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Body:          []byte(issueWebhookPayload(jiraWebhookEventCreated, "SP")),
			Configuration: map[string]any{},
			Events:        events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"clientSecret": "secret",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("missing auth header -> forbidden", func(t *testing.T) {
		status, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          []byte(issueWebhookPayload(jiraWebhookEventCreated, "SP")),
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"clientSecret": "secret",
				},
			},
		})

		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, status)
	})
}

func signedJiraHeaders(t *testing.T, secret string) http.Header {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "jira",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	return http.Header{"Authorization": []string{"Bearer " + signed}}
}

func issueWebhookPayload(event, projectKey string) string {
	return `{
		"webhookEvent": "` + event + `",
		"issue": {
			"id": "10001",
			"key": "` + projectKey + `-1",
			"fields": {
				"summary": "Test issue",
				"project": {
					"id": "10000",
					"key": "` + projectKey + `",
					"name": "Test Project"
				}
			}
		}
	}`
}

func Test__OnIssue__ExampleData(t *testing.T) {
	data := (&OnIssue{}).ExampleData()
	require.NotEmpty(t, data)
	assert.Equal(t, jiraWebhookEventCreated, data["webhookEvent"])
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
