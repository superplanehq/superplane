package cursor

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__LaunchAgent__Setup(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("valid repository mode config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"prompt":     "Fix the bug",
				"sourceMode": "repository",
				"repository": "https://github.com/org/repo",
				"branch":     "main",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.NoError(t, err)
	})

	t.Run("valid PR mode config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"prompt":     "Fix the PR",
				"sourceMode": "pr",
				"prUrl":      "https://github.com/org/repo/pull/42",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.NoError(t, err)
	})

	t.Run("missing prompt -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"sourceMode": "repository",
				"repository": "https://github.com/org/repo",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt is required")
	})

	t.Run("repository mode without repository -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"prompt":     "Fix the bug",
				"sourceMode": "repository",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository URL is required")
	})

	t.Run("PR mode without prUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"prompt":     "Fix the PR",
				"sourceMode": "pr",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR URL is required")
	})

	t.Run("repository mode with non-empty repository is accepted", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		setupCtx := core.SetupContext{
			Configuration: map[string]any{
				"prompt":     "Fix the bug",
				"sourceMode": "repository",
				"repository": "not-a-url",
			},
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		}

		err := c.Setup(setupCtx)
		require.NoError(t, err)
	})
}

func Test__LaunchAgent__Execute(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("successful launch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "agent-123",
						"status": "CREATING",
						"target": {"branchName": "cursor/agent-abc123"}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: uuid.New().String(),
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		executionID := uuid.New()
		metadataCtx := &contexts.MetadataContext{}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		requestsCtx := &contexts.RequestContext{}
		webhookCtx := &contexts.NodeWebhookContext{Secret: "platform-managed-secret"}

		execCtx := core.ExecutionContext{
			ID: executionID,
			Configuration: map[string]any{
				"prompt":       "Fix the bug",
				"sourceMode":   "repository",
				"repository":   "https://github.com/org/repo",
				"branch":       "main",
				"autoCreatePr": true,
				"useCursorBot": true,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Requests:       requestsCtx,
			Webhook:        webhookCtx,
			Logger:         logrus.NewEntry(logrus.New()),
			BaseURL:        "https://superplane.example.com",
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cursor.com/v0/agents", httpContext.Requests[0].URL.String())
		assert.NotNil(t, metadataCtx.Metadata)
		assert.Equal(t, "agent-123", executionStateCtx.KVs["agent_id"])
		assert.Equal(t, "poll", requestsCtx.Action)
	})

	t.Run("missing cloud agent key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"prompt":     "Fix the bug",
				"sourceMode": "repository",
				"repository": "https://github.com/org/repo",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud agent API key is not configured")
	})
}

func Test__LaunchAgent__HandleWebhook(t *testing.T) {
	c := &LaunchAgent{}

	t.Run("successful completion webhook", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:      "agent-123",
			Status:  "FINISHED",
			PrURL:   "https://github.com/org/repo/pull/42",
			Summary: "Fixed the bug",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)
		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{
					ID:     "agent-123",
					Status: "RUNNING",
				},
				Target: &TargetMetadata{
					BranchName: "cursor/agent-abc123",
				},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{
				"agent_id": "agent-123",
			},
		}

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

		// Verify emit was called on default channel
		assert.Equal(t, LaunchAgentDefaultChannel, executionStateCtx.Channel)
		assert.Equal(t, LaunchAgentPayloadType, executionStateCtx.Type)

		// Verify metadata was updated
		updatedMetadata := metadataCtx.Metadata.(LaunchAgentExecutionMetadata)
		assert.Equal(t, "FINISHED", updatedMetadata.Agent.Status)
		assert.Equal(t, "https://github.com/org/repo/pull/42", updatedMetadata.Target.PrURL)
	})

	t.Run("failed agent webhook", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			ID:      "agent-123",
			Status:  "failed",
			Summary: "Agent encountered an error",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)

		metadataCtx := &contexts.MetadataContext{
			Metadata: LaunchAgentExecutionMetadata{
				Agent: &AgentMetadata{
					ID:     "agent-123",
					Status: "RUNNING",
				},
			},
		}
		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{
				"agent_id": "agent-123",
			},
		}

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

		// Verify emit was called on default channel
		assert.Equal(t, LaunchAgentDefaultChannel, executionStateCtx.Channel)
	})

	t.Run("missing agent ID -> bad request", func(t *testing.T) {
		secret := "test-secret"
		payload := launchAgentWebhookPayload{
			Status: "FINISHED",
		}
		body, _ := json.Marshal(payload)
		signature := generateSignature(body, secret)

		webhookCtx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{signature}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Contains(t, err.Error(), "id missing")
	})

	t.Run("invalid JSON -> bad request", func(t *testing.T) {
		secret := "test-secret"
		signature := generateSignature([]byte("not json"), secret)

		webhookCtx := core.WebhookRequestContext{
			Body:    []byte("not json"),
			Headers: http.Header{LaunchAgentWebhookSignatureHeader: []string{signature}},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		}

		status, err := c.HandleWebhook(webhookCtx)
		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func Test__LaunchAgent__OutputChannels(t *testing.T) {
	c := &LaunchAgent{}
	channels := c.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, LaunchAgentDefaultChannel, channels[0].Name)
	assert.Equal(t, "Default", channels[0].Label)
}
