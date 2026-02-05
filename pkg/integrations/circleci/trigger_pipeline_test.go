package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__TriggerPipeline__Setup(t *testing.T) {
	component := TriggerPipeline{}

	t.Run("projectSlug is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: TriggerPipelineSpec{ProjectSlug: ""},
		})

		require.ErrorContains(t, err, "projectSlug is required")
	})

	t.Run("metadata already set -> returns early", func(t *testing.T) {
		testProject := &ProjectInfo{ID: "proj-123", Name: "test-project", Slug: "gh/myorg/test-project", URL: "https://app.circleci.com/pipelines/gh/myorg/test-project"}

		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerPipelineNodeMetadata{
				Project: testProject,
			},
		}

		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: TriggerPipelineSpec{ProjectSlug: "gh/myorg/test-project"},
		})

		require.NoError(t, err)
		metadata := metadataCtx.Get().(TriggerPipelineNodeMetadata)
		assert.Equal(t, testProject, metadata.Project)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("successful setup with new project", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"proj-123","slug":"gh/myorg/test-project","name":"test-project"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Metadata:      metadataCtx,
			Configuration: TriggerPipelineSpec{ProjectSlug: "gh/myorg/test-project"},
		})

		require.NoError(t, err)
		metadata := metadataCtx.Get().(TriggerPipelineNodeMetadata)
		assert.NotNil(t, metadata.Project)
		assert.Equal(t, "proj-123", metadata.Project.ID)
		assert.Equal(t, "gh/myorg/test-project", metadata.Project.Slug)
	})
}

func Test__TriggerPipeline__HandleWebhook(t *testing.T) {
	component := &TriggerPipeline{}

	t.Run("no circleci-signature -> 403", func(t *testing.T) {
		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature header")
	})

	t.Run("circleci-signature without v1= prefix -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("circleci-signature", "invalidsignature")

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Webhook: &contexts.WebhookContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("circleci-signature", "v1=invalidsignature")

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"type":"workflow-completed","workflow":{"status":"success"},"pipeline":{"id":"pipeline-123"}}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature but missing pipeline ID -> 400", func(t *testing.T) {
		body := []byte(`{"type":"workflow-completed","workflow":{"status":"success"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, fmt.Errorf("not found")
			},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "pipeline data missing")
	})

	t.Run("valid signature with unmatched pipeline -> ignores", func(t *testing.T) {
		body := []byte(`{"type":"workflow-completed","workflow":{"id":"wf-123","status":"success"},"pipeline":{"id":"pipeline-123"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, fmt.Errorf("not found")
			},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})

	t.Run("non-workflow event -> ignores", func(t *testing.T) {
		body := []byte(`{"type":"job-completed","job":{"status":"success"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})
}

func Test__TriggerPipeline__OutputChannels(t *testing.T) {
	component := &TriggerPipeline{}

	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 2)
	assert.Equal(t, "success", channels[0].Name)
	assert.Equal(t, "failed", channels[1].Name)
}

func Test__TriggerPipeline__Actions(t *testing.T) {
	component := &TriggerPipeline{}

	actions := component.Actions()

	assert.Len(t, actions, 2)
	assert.Equal(t, "poll", actions[0].Name)
	assert.False(t, actions[0].UserAccessible)
	assert.Equal(t, "finish", actions[1].Name)
	assert.True(t, actions[1].UserAccessible)
}
