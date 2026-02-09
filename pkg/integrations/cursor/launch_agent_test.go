package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__LaunchCloudAgent__HandleWebhook(t *testing.T) {
	component := &LaunchCloudAgent{}

	t.Run("missing signature -> 403", func(t *testing.T) {
		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("valid signature + FINISHED -> emits", func(t *testing.T) {
		secret := "test-secret"

		payload := AgentStatusWebhook{
			Event:     webhookEventStatusChange,
			ID:        "agent_1",
			Status:    agentStatusFinished,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Source:    AgentSource{Repository: "https://github.com/acme/widgets", Ref: "main"},
			Target:    &AgentTarget{URL: "https://github.com/acme/widgets", PRURL: "https://github.com/acme/widgets/pull/1"},
			Summary:   "done",
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Webhook-Signature", "sha256="+signature)

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{Metadata: LaunchCloudAgentExecutionMetadata{
			Agent:         &LaunchAgentResponse{ID: "agent_1"},
			WebhookSecret: secret,
		}}

		execCtx := &core.ExecutionContext{
			ID:             uuid.New(),
			WorkflowID:     uuid.New().String(),
			NodeID:         "n1",
			Metadata:       metadataCtx,
			ExecutionState: executionState,
		}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				require.Equal(t, "agent_id", key)
				require.Equal(t, "agent_1", value)
				return execCtx, nil
			},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.True(t, executionState.IsFinished())
		assert.Equal(t, AgentCompletedPayloadType, executionState.Type)

		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		agent, ok := data["agent"].(*LaunchAgentResponse)
		require.True(t, ok)
		assert.Equal(t, agentStatusFinished, agent.Status)
		assert.Equal(t, payload.Summary, agent.Summary)
		require.NotNil(t, agent.Target)
		assert.Equal(t, payload.Target.PRURL, agent.Target.PRURL)
		assert.Equal(t, payload.Target.URL, agent.Target.URL)
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		payload := AgentStatusWebhook{
			Event:  webhookEventStatusChange,
			ID:     "agent_2",
			Status: agentStatusFinished,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		headers := http.Header{}
		headers.Set("X-Webhook-Signature", "sha256=deadbeef")

		execCtx := &core.ExecutionContext{
			ID:         uuid.New(),
			WorkflowID: uuid.New().String(),
			NodeID:     "n1",
			Metadata: &contexts.MetadataContext{Metadata: LaunchCloudAgentExecutionMetadata{
				Agent:         &LaunchAgentResponse{ID: "agent_2"},
				WebhookSecret: secret,
			}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return execCtx, nil
			},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("valid signature + ERROR -> fails", func(t *testing.T) {
		secret := "test-secret"

		payload := AgentStatusWebhook{
			Event:  webhookEventStatusChange,
			ID:     "agent_3",
			Status: agentStatusError,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Webhook-Signature", "sha256="+signature)

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{Metadata: LaunchCloudAgentExecutionMetadata{
			Agent:         &LaunchAgentResponse{ID: "agent_3"},
			WebhookSecret: secret,
		}}

		execCtx := &core.ExecutionContext{
			ID:             uuid.New(),
			WorkflowID:     uuid.New().String(),
			NodeID:         "n1",
			Metadata:       metadataCtx,
			ExecutionState: executionState,
		}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return execCtx, nil
			},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.True(t, executionState.IsFinished())
		assert.False(t, executionState.Passed)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, executionState.FailureReason)
	})
}
