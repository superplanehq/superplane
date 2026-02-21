package sendgrid

import (
	"encoding/json"
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

func Test__SendGrid_CreateOrUpdateContact__Setup(t *testing.T) {
	component := &CreateOrUpdateContact{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing email -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"firstName": "Jane",
			},
		})

		require.ErrorContains(t, err, "email is required")
	})

	t.Run("invalid email -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"email": "not-an-email",
			},
		})

		require.ErrorContains(t, err, "invalid 'email' address")
	})

	t.Run("expression email -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    metadata,
			Configuration: map[string]any{
				"email": "{{ $[\"start 2\"].email }}",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(CreateOrUpdateContactMetadata)
		require.True(t, ok)
		assert.Equal(t, "{{ $[\"start 2\"].email }}", stored.Email)
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    metadata,
			Configuration: map[string]any{
				"email": "recipient@example.com",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(CreateOrUpdateContactMetadata)
		require.True(t, ok)
		assert.Equal(t, "recipient@example.com", stored.Email)
	})
}

func Test__SendGrid_CreateOrUpdateContact__Execute(t *testing.T) {
	component := &CreateOrUpdateContact{}

	t.Run("missing email -> fails execution and emits failed channel", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "sg-test",
				},
			},
			ExecutionState: execState,
			Configuration:  map[string]any{},
		})

		require.NoError(t, err)
		assert.Equal(t, UpsertContactFailedChannel, execState.Channel)
		assert.Equal(t, UpsertContactFailedPayloadType, execState.Type)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, execState.FailureReason)
		assert.Equal(t, "email is required", execState.FailureMessage)
	})

	t.Run("valid configuration -> sends upsert and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Status:     http.StatusText(http.StatusAccepted),
					Body:       io.NopCloser(strings.NewReader(`{"job_id":"job-123"}`)),
					Header:     http.Header{},
					Request:    &http.Request{},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sg-test",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"email":     "recipient@example.com",
				"firstName": "Jane",
				"lastName":  "Doe",
				"listIds":   []string{"list-1", "list-2"},
				"customFields": map[string]any{
					"company": "Acme",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, UpsertContactPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)

		require.Len(t, httpCtx.Requests, 1)
		body, err := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, []any{"list-1", "list-2"}, payload["list_ids"])
		contacts := payload["contacts"].([]any)
		require.Len(t, contacts, 1)
		contact := contacts[0].(map[string]any)
		assert.Equal(t, "recipient@example.com", contact["email"])
		assert.Equal(t, "Jane", contact["first_name"])
		assert.Equal(t, "Doe", contact["last_name"])
		customFields := contact["custom_fields"].(map[string]any)
		assert.Equal(t, "Acme", customFields["company"])
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
				"apiKey": "sg-invalid",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"email": "recipient@example.com",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, UpsertContactFailedChannel, execState.Channel)
		assert.Equal(t, UpsertContactFailedPayloadType, execState.Type)
	})
}
