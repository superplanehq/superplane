package gitlab

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const jobBody = `{
	"object_kind": "build",
	"ref": "main",
	"sha": "f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55",
	"build_id": 998876,
	"build_name": "unit-tests",
	"build_stage": "test",
	"build_status": "success",
	"pipeline_id": 12345,
	"project_id": 987
}`

func onJobRequest(body string, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	return core.WebhookRequestContext{
		Headers:       gitlabHeaders("Job Hook", "token"),
		Body:          []byte(body),
		Configuration: config,
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	}
}

func Test__OnJob__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnJob{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnJob__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "token"),
		Body:          []byte(jobBody),
		Configuration: map[string]any{"project": "123"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__MissingToken(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Job Hook", ""),
		Body:          []byte(jobBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "missing X-Gitlab-Token header")
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Job Hook", "wrong"),
		Body:          []byte(jobBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__MissingStatus(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		`{"object_kind":"build","build_name":"unit-tests"}`,
		map[string]any{"project": "123", "statuses": []string{"success"}},
		events,
	))

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "build_status missing")
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__StatusMatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{"project": "123", "statuses": []string{"success", "failed"}},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.job", events.Payloads[0].Type)
}

func Test__OnJob__HandleWebhook__StatusMismatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{"project": "123", "statuses": []string{"failed"}},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__NameMatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"names": []configuration.Predicate{
				{Type: configuration.PredicateTypeMatches, Value: "^unit-.*"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.job", events.Payloads[0].Type)
}

func Test__OnJob__HandleWebhook__NameMismatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"names": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "deploy"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__RefMatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"refs": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "main"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.job", events.Payloads[0].Type)
}

func Test__OnJob__HandleWebhook__RefMismatch(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"refs": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "release"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnJob__HandleWebhook__NoFiltersEmitsEveryJob(t *testing.T) {
	trigger := &OnJob{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onJobRequest(
		jobBody,
		map[string]any{"project": "123"},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.job", events.Payloads[0].Type)
}

func Test__OnJob__Setup(t *testing.T) {
	trigger := &OnJob{}
	metadata := Metadata{
		Projects: []ProjectMetadata{
			{ID: 123, Name: "group/example", URL: "https://gitlab.com/group/example"},
		},
	}

	t.Run("project is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": ""},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("project is not accessible", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "456"},
		})

		require.ErrorContains(t, err, "project 456 is not accessible to integration")
	})

	t.Run("metadata is set and job webhook is requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Metadata: metadata}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "123", "statuses": []string{"success"}},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "job", webhookConfig.EventType)
		assert.Equal(t, "123", webhookConfig.ProjectID)
	})
}

// The "job" event type is what turns on job_events when SuperPlane creates the
// GitLab hook, so assert it reaches the API request rather than just the switch.
func Test__GitLabWebhookHandler__Setup__JobEvents(t *testing.T) {
	handler := &GitLabWebhookHandler{}
	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			GitlabMockResponse(http.StatusCreated, `{"id": 42, "job_events": true}`),
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP: mockHTTP,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"accessToken": "pat-123",
				"baseUrl":     "https://gitlab.com",
			},
		},
		Webhook: &contexts.WebhookContext{
			URL:           "https://superplane.example.com/webhook",
			Secret:        []byte("token"),
			Configuration: WebhookConfiguration{EventType: "job", ProjectID: "123"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, &WebhookMetadata{ID: 42}, metadata)

	require.Len(t, mockHTTP.Requests, 1)
	body, err := io.ReadAll(mockHTTP.Requests[0].Body)
	require.NoError(t, err)

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, true, payload["job_events"])
	assert.Equal(t, false, payload["pipeline_events"])
}
