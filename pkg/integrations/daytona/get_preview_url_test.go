package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__PreviewURL__Setup(t *testing.T) {
	component := GetPreviewURLComponent{}

	t.Run("valid setup with defaults", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing sandbox -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   appCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "sandbox is required")
	})

	t.Run("invalid port -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
				"port":    70000,
			},
		})

		require.ErrorContains(t, err, "port must be between")
	})

	t.Run("invalid expiresInSeconds for signed URL -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox":          "sandbox-123",
				"signed":           true,
				"expiresInSeconds": 86401,
			},
		})

		require.ErrorContains(t, err, "expiresInSeconds must be between")
	})
}

func Test__PreviewURL__Execute(t *testing.T) {
	component := GetPreviewURLComponent{}

	t.Run("signed preview URL generation -> emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"sandboxId":"sandbox-123","port":3000,"token":"signed-token-abc","url":"https://3000-signed-token-abc.preview.daytona.app"}`,
					)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandbox":          "sandbox-123",
				"port":             3000,
				"signed":           true,
				"expiresInSeconds": 3600,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, PreviewURLPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		wrappedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		payloadData, ok := wrappedPayload["data"].(PreviewURLPayload)
		require.True(t, ok)

		assert.Equal(t, "sandbox-123", payloadData.Sandbox)
		assert.Equal(t, 3000, payloadData.Port)
		assert.True(t, payloadData.Signed)
		assert.Equal(t, "signed-token-abc", payloadData.Token)
		assert.Equal(t, "https://3000-signed-token-abc.preview.daytona.app", payloadData.URL)
		assert.Equal(t, 3600, payloadData.ExpiresInSeconds)
	})

	t.Run("standard preview URL generation -> emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"sandboxId":"sandbox-123","token":"header-token-xyz","url":"https://3000-sandbox-123.preview.daytona.app"}`,
					)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
				"port":    3000,
				"signed":  false,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		require.Len(t, execCtx.Payloads, 1)

		wrappedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		payloadData, ok := wrappedPayload["data"].(PreviewURLPayload)
		require.True(t, ok)

		assert.False(t, payloadData.Signed)
		assert.Equal(t, "header-token-xyz", payloadData.Token)
		assert.Equal(t, "https://3000-sandbox-123.preview.daytona.app", payloadData.URL)
		assert.Zero(t, payloadData.ExpiresInSeconds)
	})

	t.Run("signed preview URL generation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message":"preview failed"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
				"signed":  true,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to generate signed preview URL")
	})

	t.Run("standard preview URL generation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message":"preview failed"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
				"signed":  false,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to generate preview URL")
	})
}

func Test__PreviewURL__ComponentInfo(t *testing.T) {
	component := GetPreviewURLComponent{}

	assert.Equal(t, "daytona.getPreviewUrl", component.Name())
	assert.Equal(t, "Get Preview URL", component.Label())
	assert.Equal(t, "Generate a preview URL for a sandbox port", component.Description())
	assert.Equal(t, "daytona", component.Icon())
	assert.Equal(t, "orange", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__PreviewURL__Configuration(t *testing.T) {
	component := GetPreviewURLComponent{}

	config := component.Configuration()
	assert.Len(t, config, 4)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "sandbox")
	assert.Contains(t, fieldNames, "port")
	assert.Contains(t, fieldNames, "signed")
	assert.Contains(t, fieldNames, "expiresInSeconds")

	var sandboxFieldType string
	for _, field := range config {
		if field.Name == "sandbox" {
			sandboxFieldType = field.Type
			break
		}
	}
	assert.Equal(t, configuration.FieldTypeIntegrationResource, sandboxFieldType)
}

func Test__PreviewURL__OutputChannels(t *testing.T) {
	component := GetPreviewURLComponent{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
