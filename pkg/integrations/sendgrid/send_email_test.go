package sendgrid

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendGrid_SendEmail__Setup(t *testing.T) {
	component := &SendEmail{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing to -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"subject": "Test Subject",
				"body":    "Test body",
			},
		})

		require.ErrorContains(t, err, "to is required")
	})

	t.Run("missing subject -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":   "test@example.com",
				"body": "Test body",
				"mode": "text",
			},
		})

		require.ErrorContains(t, err, "subject is required")
	})

	t.Run("missing body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":      "test@example.com",
				"subject": "Test Subject",
				"mode":    "text",
			},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("template mode missing templateId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":   "test@example.com",
				"mode": "template",
			},
		})

		require.ErrorContains(t, err, "templateId is required")
	})

	t.Run("invalid email in to -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":      "not-an-email",
				"subject": "Test Subject",
				"body":    "Test body",
				"mode":    "text",
			},
		})

		require.ErrorContains(t, err, "invalid 'to' email addresses")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    metadata,
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Hello, this is a test.",
				"mode":    "text",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendEmailMetadata)
		require.True(t, ok)
		assert.Equal(t, []string{"recipient@example.com"}, stored.To)
		assert.Equal(t, "Test Subject", stored.Subject)
	})
}

func Test__SendGrid_SendEmail__Execute(t *testing.T) {
	component := &SendEmail{}

	t.Run("missing to -> fails execution and emits failed channel", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey":    "sg-test",
					"fromEmail": "sender@example.com",
				},
			},
			ExecutionState: execState,
			Configuration: map[string]any{
				"subject": "Test",
				"body":    "Body",
				"mode":    "text",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, SendEmailFailedChannel, execState.Channel)
		assert.Equal(t, SendEmailFailedPayloadType, execState.Type)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, execState.FailureReason)
		assert.Equal(t, "to is required", execState.FailureMessage)
	})

	t.Run("valid configuration -> sends email and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Status:     http.StatusText(http.StatusAccepted),
					Header: http.Header{
						"X-Message-Id": []string{"msg-123"},
					},
					Body:    io.NopCloser(strings.NewReader("")),
					Request: &http.Request{},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":    "sg-test",
				"fromEmail": "sender@example.com",
				"fromName":  "Sender",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Hello!",
				"mode":    "text",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, SendEmailPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
	})

	t.Run("template mode -> sends template email", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Status:     http.StatusText(http.StatusAccepted),
					Header: http.Header{
						"X-Message-Id": []string{"msg-456"},
					},
					Body:    io.NopCloser(strings.NewReader("")),
					Request: &http.Request{},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":    "sg-test",
				"fromEmail": "sender@example.com",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"to":         "recipient@example.com",
				"mode":       "template",
				"templateId": "d-1234567890abcdef",
				"templateData": map[string]any{
					"name": "Jane",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, SendEmailPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
	})

	t.Run("SendGrid rejection -> emits failed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Status:     http.StatusText(http.StatusUnauthorized),
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"message":"invalid"}]}`)),
					Header:     http.Header{},
					Request:    &http.Request{},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":    "sg-invalid",
				"fromEmail": "sender@example.com",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Hello!",
				"mode":    "text",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, SendEmailFailedChannel, execState.Channel)
		assert.Equal(t, SendEmailFailedPayloadType, execState.Type)
	})
}
