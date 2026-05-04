package render

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Render_RetrieveCustomDomain__Setup(t *testing.T) {
	component := &RetrieveCustomDomain{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"domainName": "app.example.com"},
		})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing domainName -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123"},
		})
		require.ErrorContains(t, err, "domainName is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "domainName": "app.example.com"},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration -> stores service metadata when context is available", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"srv-123","name":"backend-api"}`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"service":    "srv-123",
				"domainName": "app.example.com",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, RenderServiceNodeMetadata{
			Service: &RenderServiceMetadata{ID: "srv-123", Name: "backend-api"},
		}, metadataCtx.Get())
	})
}

func Test__Render_RetrieveCustomDomain__Execute(t *testing.T) {
	component := &RetrieveCustomDomain{}

	t.Run("valid configuration -> retrieves custom domain and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"verified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service":    "srv-123",
				"domainName": "app.example.com",
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, RetrieveCustomDomainPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "cdm-abc123", data["id"])
		assert.Equal(t, "app.example.com", data["name"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "verified", data["verificationStatus"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/custom-domains/app.example.com")
	})
}
