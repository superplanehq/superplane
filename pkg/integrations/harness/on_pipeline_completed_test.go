package harness

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type failingNodeWebhookContext struct {
	secret string
	err    error
}

func (c *failingNodeWebhookContext) Setup() (string, error) {
	return "", nil
}

func (c *failingNodeWebhookContext) GetSecret() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(c.secret), nil
}

func (c *failingNodeWebhookContext) SetSecret(secret []byte) error {
	if c.err != nil {
		return c.err
	}
	c.secret = string(secret)
	return nil
}

func (c *failingNodeWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return []byte(c.secret), []byte(c.secret), nil
}

func (c *failingNodeWebhookContext) GetBaseURL() string {
	return "http://localhost:3000/api/v1"
}

func Test__OnPipelineCompleted__Setup(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("without pipeline filter -> polling mode", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}}
		requestCtx := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"organization":{"identifier":"default","name":"Default"}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"projectResponse":{"project":{"identifier":"default_project","name":"Default Project"}}}]}}`,
					)),
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:     "default",
				ProjectID: "default_project",
				Statuses:  []string{"failed"},
			},
			HTTP:        httpCtx,
			Metadata:    metadataCtx,
			Webhook:     &contexts.WebhookContext{},
			Integration: integrationCtx,
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Empty(t, metadata.PipelineIdentifier)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		requestConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Empty(t, requestConfig.PipelineIdentifier)
		assert.Equal(t, "default", requestConfig.OrgID)
		assert.Equal(t, "default_project", requestConfig.ProjectID)
		assert.Equal(t, OnPipelineCompletedPollAction, requestCtx.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requestCtx.Duration)
	})

	t.Run("with pipeline filter -> webhook requested and poll fallback scheduled", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}}
		requestCtx := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"organization":{"identifier":"default","name":"Default"}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"projectResponse":{"project":{"identifier":"default_project","name":"Default Project"}}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"yamlPipeline":"pipeline:\n  identifier: deploy\n"}}`,
					)),
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:        httpCtx,
			Metadata:    metadataCtx,
			Webhook:     &contexts.WebhookContext{},
			Integration: integrationCtx,
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		_, ok := metadataCtx.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		requestConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "deploy", requestConfig.PipelineIdentifier)
		assert.Equal(t, "default", requestConfig.OrgID)
		assert.Equal(t, "default_project", requestConfig.ProjectID)
		assert.Equal(t, []string{"PipelineEnd"}, requestConfig.EventTypes)
		assert.Equal(t, OnPipelineCompletedPollAction, requestCtx.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requestCtx.Duration)
	})

	t.Run("invalid pipeline selection fails setup", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}}
		requestCtx := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"organization":{"identifier":"default","name":"Default"}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"projectResponse":{"project":{"identifier":"default_project","name":"Default Project"}}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"pipeline not found"}`)),
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "missing",
				Statuses:           []string{"failed"},
			},
			HTTP:        httpCtx,
			Metadata:    metadataCtx,
			Webhook:     &contexts.WebhookContext{},
			Integration: integrationCtx,
			Requests:    requestCtx,
		})

		require.ErrorContains(t, err, `pipeline "missing" not found or inaccessible in organization "default" project "default_project"`)
		require.Empty(t, integrationCtx.WebhookRequests)
		assert.Empty(t, requestCtx.Action)
	})
}

func Test__OnPipelineCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("unauthorized request returns forbidden before config validation", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-auth","pipelineIdentifier":"deploy","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			// Intentionally invalid/missing config.
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook authorization")
	})

	t.Run("invalid webhook secret -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer wrong")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Events: &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook authorization")
	})

	t.Run("emits event when status and pipeline match", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, OnPipelineCompletedPayloadType, events.Payloads[0].Type)

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-1", storedMetadata.LastTimestamplessExecutionID)
		assert.Equal(t, "", storedMetadata.LastExecutionID)
	})

	t.Run("treats errored webhook status as failed terminal", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-errored","pipelineIdentifier":"deploy","status":"ERRORED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		eventData, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "failed", eventData["status"])
	})

	t.Run("without webhook secret rejects request", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-2","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: ""},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Events: events,
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "webhook secret is not configured")
		assert.Equal(t, 0, events.Count())
	})

	t.Run("secret retrieval failure -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-2","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Webhook: &failingNodeWebhookContext{err: errors.New("secret backend unavailable")},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Events: &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "failed to read webhook secret")
	})

	t.Run("missing webhook context -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-2","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Events: &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "webhook context is required")
	})

	t.Run("ignores non pipeline completed event types", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"STAGE_END","data":{"planExecutionId":"exec-3","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("webhook event without end timestamp is still accepted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Minute).UnixMilli(),
			LastExecutionID:    "exec-prev",
		}}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-no-end","pipelineIdentifier":"deploy","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("webhook without timestamp is not re-emitted by poll for same execution", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		oldCheckpoint := time.Now().Add(-10 * time.Minute).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: oldCheckpoint,
			LastExecutionID:    "exec-prev",
		}}

		webhookCode, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-no-end","pipelineIdentifier":"deploy","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})
		require.NoError(t, webhookErr)
		assert.Equal(t, http.StatusOK, webhookCode)
		assert.Equal(t, 1, events.Count())

		endedTs := time.Now().Add(-5 * time.Minute).UnixMilli()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-no-end","pipelineIdentifier":"deploy","status":"FAILED","endTs":"%d"}]}}`,
							endedTs,
						),
					)),
				},
			},
		}
		requests := &contexts.RequestContext{}

		_, pollErr := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "pat.acc-123.test",
				},
			},
		})
		require.NoError(t, pollErr)
		assert.Equal(t, 1, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-no-end", storedMetadata.LastExecutionID)
		assert.EqualValues(t, endedTs, storedMetadata.LastExecutionEnded)
	})

	t.Run("status-filtered webhook updates checkpoint to avoid reprocessing", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		endTimestamp := time.Now().Add(time.Minute).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Minute).UnixMilli(),
			LastExecutionID:    "exec-prev",
		}}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body: []byte(fmt.Sprintf(
				`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-filtered","pipelineIdentifier":"deploy","nodeStatus":"SUCCESS","endTs":"%d"}}`,
				endTimestamp,
			)),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-filtered", storedMetadata.LastExecutionID)
		assert.EqualValues(t, endTimestamp, storedMetadata.LastExecutionEnded)
	})

	t.Run("ignores duplicate webhook executions already checkpointed", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}
		now := time.Now().UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionID:    "exec-4",
			LastExecutionEnded: now,
		}}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body: []byte(fmt.Sprintf(
				`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-4","pipelineIdentifier":"deploy","nodeStatus":"FAILED","endTs":%d}}`,
				now/1000,
			)),
			Webhook: &contexts.WebhookContext{Secret: "expected"},
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			Metadata: metadata,
			Events:   events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("poll emits matching completion events and updates checkpoint", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		now := time.Now().UnixMilli()
		endTs := now + int64(time.Minute/time.Millisecond)
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-5 * time.Minute).UnixMilli(),
			LastExecutionID:    "exec-old",
		}}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"SUCCESS","endTs":"%d"}]}}`,
							endTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:     "default",
				ProjectID: "default_project",
				Statuses:  []string{"succeeded"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, OnPipelineCompletedPayloadType, events.Payloads[0].Type)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-1", storedMetadata.LastExecutionID)
	})

	t.Run("poll defers recent executions in webhook mode to avoid duplicates", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		oldCheckpoint := time.Now().Add(-time.Hour).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Hour).UnixMilli(),
			LastExecutionID:    "exec-old",
		}}
		recentEndTs := time.Now().UnixMilli()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-recent","pipelineIdentifier":"deploy","status":"FAILED","endTs":"%d"}]}}`,
							recentEndTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)
		assert.Equal(t, 0, events.Count())
		assert.Equal(t, 1, len(httpCtx.Requests))

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-old", storedMetadata.LastExecutionID)
		assert.EqualValues(t, oldCheckpoint, storedMetadata.LastExecutionEnded)
	})

	t.Run("poll stops batch after race-window deferral to avoid checkpoint skipping deferred events", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		oldCheckpoint := time.Now().Add(-time.Hour).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: oldCheckpoint,
			LastExecutionID:    "exec-old",
		}}

		deferredEndTs := time.Now().Add(-30 * time.Second).UnixMilli()
		filteredEndTs := time.Now().Add(-10 * time.Second).UnixMilli()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-filtered","pipelineIdentifier":"deploy","status":"SUCCESS","endTs":"%d"},{"planExecutionId":"exec-deferred","pipelineIdentifier":"deploy","status":"FAILED","endTs":"%d"}]}}`,
							filteredEndTs,
							deferredEndTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-old", storedMetadata.LastExecutionID)
		assert.EqualValues(t, oldCheckpoint, storedMetadata.LastExecutionEnded)
	})

	t.Run("poll checkpoints recent executions when they are filtered out by status", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Hour).UnixMilli(),
			LastExecutionID:    "exec-old",
		}}
		recentEndTs := time.Now().UnixMilli()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-recent-filtered","pipelineIdentifier":"deploy","status":"FAILED","endTs":"%d"}]}}`,
							recentEndTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"succeeded"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)
		assert.Equal(t, 0, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-recent-filtered", storedMetadata.LastExecutionID)
		assert.EqualValues(t, recentEndTs, storedMetadata.LastExecutionEnded)
	})

	t.Run("poll does not checkpoint non-terminal executions", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		oldCheckpoint := time.Now().Add(-time.Hour).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: oldCheckpoint,
			LastExecutionID:    "exec-old",
		}}
		startTs := time.Now().UnixMilli()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-running","pipelineIdentifier":"deploy","status":"RUNNING","startTs":"%d"}]}}`,
							startTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)
		assert.Equal(t, 0, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-old", storedMetadata.LastExecutionID)
		assert.EqualValues(t, oldCheckpoint, storedMetadata.LastExecutionEnded)
	})

	t.Run("poll retries transient API errors without failing the action", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		checkpointTime := time.Now().Add(-time.Hour).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: checkpointTime,
			LastExecutionID:    "exec-old",
		}}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{"message":"temporary outage"}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:     "default",
				ProjectID: "default_project",
				Statuses:  []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)
		assert.Equal(t, 0, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, "exec-old", storedMetadata.LastExecutionID)
		assert.EqualValues(t, checkpointTime, storedMetadata.LastExecutionEnded)
		assert.Equal(t, 1, storedMetadata.PollErrorCount)
	})

	t.Run("poll falls back to unfiltered list when pipelineIdentifier filter is unsupported", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		endTs := time.Now().Add(-10 * time.Minute).UnixMilli()
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Hour).UnixMilli(),
			LastExecutionID:    "exec-old",
		}}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"code":400,"message":"Unknown field pipelineIdentifier in request payload"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						fmt.Sprintf(
							`{"data":{"content":[{"planExecutionId":"exec-keep","pipelineIdentifier":"deploy","status":"FAILED","endTs":"%d"},{"planExecutionId":"exec-skip","pipelineIdentifier":"other","status":"FAILED","endTs":"%d"}]}}`,
							endTs,
							endTs,
						),
					)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy",
				Statuses:           []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		firstBody, firstErr := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, firstErr)
		assert.Contains(t, string(firstBody), `"pipelineIdentifier":"deploy"`)
		secondBody, secondErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, secondErr)
		assert.NotContains(t, string(secondBody), `"pipelineIdentifier"`)
		assert.Equal(t, 1, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.True(t, storedMetadata.DisableServerPipelineIDFilterInAPI)
		assert.Equal(t, "exec-skip", storedMetadata.LastExecutionID)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
	})

	t.Run("poll keeps scheduling after max consecutive errors", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{Metadata: OnPipelineCompletedMetadata{
			LastExecutionEnded: time.Now().Add(-time.Hour).UnixMilli(),
			LastExecutionID:    "exec-old",
			PollErrorCount:     OnPipelineCompletedMaxPollErrors - 1,
		}}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{"message":"temporary outage"}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: OnPipelineCompletedPollAction,
			Configuration: OnPipelineCompletedConfiguration{
				OrgID:     "default",
				ProjectID: "default_project",
				Statuses:  []string{"failed"},
			},
			HTTP:     httpCtx,
			Metadata: metadata,
			Requests: requests,
			Events:   events,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "pat.acc-123.test",
					"orgId":     "default",
					"projectId": "default_project",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, OnPipelineCompletedPollAction, requests.Action)
		assert.Equal(t, OnPipelineCompletedPollInterval, requests.Duration)
		assert.Equal(t, 0, events.Count())

		storedMetadata, ok := metadata.Get().(OnPipelineCompletedMetadata)
		require.True(t, ok)
		assert.Equal(t, OnPipelineCompletedMaxPollErrors, storedMetadata.PollErrorCount)
	})
}

func Test__ParseEpochMilliseconds(t *testing.T) {
	now := time.Now().UnixMilli()
	nowRoundedToSecond := now - (now % 1000)
	nowSeconds := now / 1000
	rfc3339 := time.UnixMilli(nowRoundedToSecond).UTC().Format(time.RFC3339)
	textual := time.UnixMilli(nowRoundedToSecond).UTC().Format("Mon Jan 2 15:04:05 MST 2006")

	assert.EqualValues(t, nowRoundedToSecond, parseEpochMilliseconds(strconv.FormatInt(nowSeconds, 10)))
	assert.EqualValues(t, now, parseEpochMilliseconds(strconv.FormatInt(now, 10)))
	assert.EqualValues(t, nowRoundedToSecond, parseEpochMilliseconds(rfc3339))
	assert.EqualValues(t, nowRoundedToSecond, parseEpochMilliseconds(textual))
	assert.EqualValues(t, 0, parseEpochMilliseconds("not-a-time"))
}

func Test__OnPipelineCompleted__CheckpointHelpers(t *testing.T) {
	t.Run("updateCheckpoint does not regress execution id without timestamp", func(t *testing.T) {
		metadata := OnPipelineCompletedMetadata{
			LastExecutionID:    "zzz",
			LastExecutionEnded: 0,
		}
		execution := ExecutionSummary{ExecutionID: "aaa"}

		updated := updateCheckpoint(metadata, execution)
		assert.Equal(t, "zzz", updated.LastExecutionID)
		assert.EqualValues(t, 0, updated.LastExecutionEnded)
		assert.Equal(t, "aaa", updated.LastTimestamplessExecutionID)
	})

	t.Run("isNewerExecution dedupes same execution id even when timestamp becomes available", func(t *testing.T) {
		endTs := time.Now().UnixMilli()
		metadata := OnPipelineCompletedMetadata{
			LastExecutionID:    "exec-1",
			LastExecutionEnded: endTs - 60_000,
		}
		execution := ExecutionSummary{
			ExecutionID: "exec-1",
			EndedAt:     strconv.FormatInt(endTs, 10),
		}

		assert.False(t, isNewerExecution(metadata, execution))
	})

	t.Run("isNewerExecution allows checkpoint refresh for execution seen first without timestamp", func(t *testing.T) {
		endTs := time.Now().UnixMilli()
		metadata := OnPipelineCompletedMetadata{
			LastExecutionID:              "exec-prev",
			LastExecutionEnded:           endTs - 60_000,
			LastTimestamplessExecutionID: "exec-1",
		}
		execution := ExecutionSummary{
			ExecutionID: "exec-1",
			EndedAt:     strconv.FormatInt(endTs, 10),
		}

		assert.True(t, isNewerExecution(metadata, execution))
	})
}
