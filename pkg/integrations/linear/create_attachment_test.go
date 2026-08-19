package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func attachmentConfig() map[string]any {
	return map[string]any{
		"issue": "ENG-142",
		"title": "Deploy - staging",
		"url":   "https://app.superplane.com/runs/8f21c4d0",
	}
}

func Test__CreateAttachment__Setup(t *testing.T) {
	component := CreateAttachment{}

	t.Run("missing issue -> error", func(t *testing.T) {
		config := attachmentConfig()
		delete(config, "issue")

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("blank issue -> error", func(t *testing.T) {
		config := attachmentConfig()
		config["issue"] = "   "

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("missing title -> error", func(t *testing.T) {
		config := attachmentConfig()
		delete(config, "title")

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("missing url -> error", func(t *testing.T) {
		config := attachmentConfig()
		delete(config, "url")

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "url is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: attachmentConfig(),
		})

		require.NoError(t, err)
	})
}

func Test__CreateAttachment__Execute(t *testing.T) {
	component := CreateAttachment{}

	t.Run("emits the created attachment", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentCreate":{"success":true,"attachment":{"id":"a1","title":"Deploy - staging","subtitle":"Succeeded in 4m 12s","url":"https://app.superplane.com/runs/8f21c4d0","issue":{"id":"i1","identifier":"ENG-142","title":"Boom","url":"https://linear.app/acme/issue/ENG-142"}}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  attachmentConfig(),
		})

		require.NoError(t, err)
		assert.Equal(t, AttachmentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		attachment, ok := wrapped["data"].(*Attachment)
		require.True(t, ok)
		assert.Equal(t, "a1", attachment.ID)
		assert.Equal(t, "https://app.superplane.com/runs/8f21c4d0", attachment.URL)
		require.NotNil(t, attachment.Issue)
		assert.Equal(t, "ENG-142", attachment.Issue.Identifier)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", input["issueId"])
		assert.Equal(t, "Deploy - staging", input["title"])
		assert.Equal(t, "https://app.superplane.com/runs/8f21c4d0", input["url"])
	})

	t.Run("optional fields are omitted when blank", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentCreate":{"success":true,"attachment":{"id":"a1","title":"Deploy - staging","url":"https://app.superplane.com/runs/8f21c4d0"}}}}`),
			},
		}

		config := attachmentConfig()
		config["subtitle"] = "  "
		config["iconUrl"] = ""

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.NoError(t, err)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, input, "subtitle")
		assert.NotContains(t, input, "iconUrl")
	})

	t.Run("optional fields are sent when set", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentCreate":{"success":true,"attachment":{"id":"a1","title":"Deploy - staging","url":"https://app.superplane.com/runs/8f21c4d0"}}}}`),
			},
		}

		config := attachmentConfig()
		config["subtitle"] = "  Succeeded in 4m 12s  "
		config["iconUrl"] = "https://example.com/icon.png"

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.NoError(t, err)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Succeeded in 4m 12s", input["subtitle"])
		assert.Equal(t, "https://example.com/icon.png", input["iconUrl"])
	})

	t.Run("API failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Entity not found"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  attachmentConfig(),
		})

		require.ErrorContains(t, err, "Entity not found")
	})

	//
	// Linear takes iconUrl on AttachmentCreateInput but does not expose it on the
	// Attachment type, so selecting it fails the whole mutation with
	// `Cannot query field "iconUrl" on type "Attachment"`.
	//
	t.Run("iconUrl is sent but never selected back", func(t *testing.T) {
		assert.NotContains(t, attachmentFields, "iconUrl")

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentCreate":{"success":true,"attachment":{"id":"a1","title":"Deploy - staging","url":"https://app.superplane.com/runs/8f21c4d0"}}}}`),
			},
		}

		config := attachmentConfig()
		config["iconUrl"] = "https://example.com/icon.png"

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.NoError(t, err)

		request := variablesFromRequest(t, httpContext, 0)
		input, ok := request["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/icon.png", input["iconUrl"])
	})

	t.Run("unsuccessful response is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentCreate":{"success":false}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  attachmentConfig(),
		})

		require.ErrorContains(t, err, "not created")
	})
}
