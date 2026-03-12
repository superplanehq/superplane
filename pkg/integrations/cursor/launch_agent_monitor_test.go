package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func generateSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func Test__LaunchAgent__HandleWebhook__SignatureVerification(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("missing signature header -> unauthorized", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:     "agent-123",
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{}, // No signature header
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, status)
		assert.Contains(t, err.Error(), "missing signature header")
	})

	t.Run("invalid signature -> unauthorized", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:     "agent-123",
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{"invalid-signature"}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, status)
		assert.Contains(t, err.Error(), "invalid webhook signature")
	})

	t.Run("valid signature -> success", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:     "agent-123",
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent:  &AgentMetadata{ID: "agent-123", Status: "RUNNING"},
				Target: &TargetMetadata{BranchName: "cursor/agent-abc"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{"agent_id": "agent-123"}}

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{signature}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: executionStateCtx,
					Logger:         logrus.NewEntry(logrus.New()),
				}, nil
			},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})
}

func Test__LaunchAgent__HandleWebhook__IdempotencyCheck(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("already terminal status -> returns OK without processing", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:     "agent-123",
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{ID: "agent-123", Status: "FINISHED"}, // Already terminal
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{"agent_id": "agent-123"}}

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{signature}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: executionStateCtx,
					Logger:         logrus.NewEntry(logrus.New()),
				}, nil
			},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		// Emit should NOT have been called
		assert.Empty(t, executionStateCtx.Channel)
	})
}

func Test__LaunchAgent__HandleWebhook__ExecutionNotFound(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("execution not found -> returns OK to stop retries", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:     "agent-123",
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{signature}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, errors.New("execution not found")
			},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})
}

func Test__LaunchAgent__Actions(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("returns poll action", func(t *testing.T) {
		actions := c.Actions()
		require.Len(t, actions, 1)
		assert.Equal(t, "poll", actions[0].Name)
		assert.False(t, actions[0].UserAccessible)
	})
}

func Test__LaunchAgent__HandleAction(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("unknown action -> error", func(t *testing.T) {
		ctx := core.ActionContext{Name: "unknown"}
		err := c.HandleAction(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}

func Test__LaunchAgent__Poll(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("execution already finished -> no-op", func(t *testing.T) {
		executionStateCtx := &contexts.ExecutionStateContext{Finished: true}

		ctx := core.ActionContext{
			Name:           "poll",
			ExecutionState: executionStateCtx,
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
	})

	t.Run("no agent metadata -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{Agent: nil},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false}

		ctx := core.ActionContext{
			Name:           "poll",
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Parameters:     map[string]any{},
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
	})

	t.Run("agent already terminal -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{ID: "agent-123", Status: "FINISHED"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false}

		ctx := core.ActionContext{
			Name:           "poll",
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Parameters:     map[string]any{},
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
	})

	t.Run("max poll attempts exceeded -> emits timeout", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent:  &AgentMetadata{ID: "agent-123", Status: "RUNNING"},
				Target: &TargetMetadata{BranchName: "cursor/agent-abc"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false, KVs: map[string]string{}}

		ctx := core.ActionContext{
			Name:           "poll",
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Parameters:     map[string]any{"attempt": float64(LaunchAgentMaxPollAttempts + 1)},
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.Equal(t, LaunchAgentDefaultChannel, executionStateCtx.Channel)
	})

	t.Run("successful poll with terminal status -> emits completion", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "agent-123",
						"status": "FINISHED",
						"summary": "Task completed",
						"target": {"prUrl": "https://github.com/org/repo/pull/42"}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"launchAgentKey": "test-key"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent:  &AgentMetadata{ID: "agent-123", Status: "RUNNING"},
				Target: &TargetMetadata{BranchName: "cursor/agent-abc"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false, KVs: map[string]string{}}

		ctx := core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Parameters:     map[string]any{"attempt": float64(1)},
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.Equal(t, LaunchAgentDefaultChannel, executionStateCtx.Channel)
		assert.Equal(t, LaunchAgentPayloadType, executionStateCtx.Type)
	})

	t.Run("poll API error -> schedules next poll with error count", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"launchAgentKey": "test-key"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent:  &AgentMetadata{ID: "agent-123", Status: "RUNNING"},
				Target: &TargetMetadata{BranchName: "cursor/agent-abc"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false, KVs: map[string]string{}}
		requestsCtx := &contexts.RequestContext{}

		ctx := core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Requests:       requestsCtx,
			Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
		assert.Equal(t, "poll", requestsCtx.Action)
		assert.Equal(t, 2, requestsCtx.Params["attempt"])
		assert.Equal(t, 1, requestsCtx.Params["errors"])
	})

	t.Run("max poll errors exceeded -> emits error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"launchAgentKey": "test-key"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent:  &AgentMetadata{ID: "agent-123", Status: "RUNNING"},
				Target: &TargetMetadata{BranchName: "cursor/agent-abc"},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{Finished: false, KVs: map[string]string{}}

		ctx := core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Parameters:     map[string]any{"attempt": float64(1), "errors": float64(LaunchAgentMaxPollErrors - 1)},
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.HandleAction(ctx)
		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.Equal(t, LaunchAgentDefaultChannel, executionStateCtx.Channel)
	})
}

func Test__LaunchAgent__ScheduleNextPoll(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("calculates exponential backoff", func(t *testing.T) {
		requestsCtx := &contexts.RequestContext{}

		ctx := core.ActionContext{
			Requests: requestsCtx,
		}

		err := c.scheduleNextPoll(ctx, 2, 0)
		require.NoError(t, err)
		assert.Equal(t, "poll", requestsCtx.Action)
		assert.Equal(t, 2, requestsCtx.Params["attempt"])
		assert.Equal(t, 0, requestsCtx.Params["errors"])
		// First poll: 30s * 2^(2-1) = 60s
		assert.Equal(t, 60*time.Second, requestsCtx.Duration)
	})

	t.Run("caps at max poll interval", func(t *testing.T) {
		requestsCtx := &contexts.RequestContext{}

		ctx := core.ActionContext{
			Requests: requestsCtx,
		}

		err := c.scheduleNextPoll(ctx, 20, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, requestsCtx.Duration, LaunchAgentMaxPollInterval)
	})
}

func Test__LaunchAgent__Cancel(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("no agent metadata -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{Agent: nil},
		}

		ctx := core.ExecutionContext{
			ID:       uuid.New(),
			Metadata: metadataCtx,
		}

		err := c.Cancel(ctx)
		require.NoError(t, err)
	})

	t.Run("empty agent ID -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{ID: ""},
			},
		}

		ctx := core.ExecutionContext{
			ID:       uuid.New(),
			Metadata: metadataCtx,
		}

		err := c.Cancel(ctx)
		require.NoError(t, err)
	})

	t.Run("successful cancel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"launchAgentKey": "test-key"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{ID: "agent-123"},
			},
		}

		ctx := core.ExecutionContext{
			ID:          uuid.New(),
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Cancel(ctx)
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "agent-123")
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	})
}

func Test__LaunchAgent__Cleanup(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("returns nil (no-op)", func(t *testing.T) {
		ctx := core.SetupContext{}
		err := c.Cleanup(ctx)
		require.NoError(t, err)
	})
}
