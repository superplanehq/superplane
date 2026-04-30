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

func Test__Render_AddCustomDomain__Setup(t *testing.T) {
	component := &AddCustomDomain{}

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
}

func Test__Render_AddCustomDomain__Execute(t *testing.T) {
	component := &AddCustomDomain{}

	t.Run("waitForVerification false -> adds domain and immediately emits success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Configuration: map[string]any{
				"service":             "srv-123",
				"domainName":          "app.example.com",
				"waitForVerification": false,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, AddCustomDomainSuccessOutputChannel, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "cdm-abc123", data["id"])
		assert.Equal(t, "app.example.com", data["name"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "unverified", data["verificationStatus"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/custom-domains")
	})

	t.Run("waitForVerification true, already verified -> emits success immediately", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"verified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Configuration: map[string]any{
				"service":             "srv-123",
				"domainName":          "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, AddCustomDomainSuccessOutputChannel, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
	})

	t.Run("waitForVerification true, verification failed -> emits failed immediately", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"failed"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Configuration: map[string]any{
				"service":             "srv-123",
				"domainName":          "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, AddCustomDomainFailedOutputChannel, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
	})

	t.Run("waitForVerification true, unverified -> schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service":             "srv-123",
				"domainName":          "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, AddCustomDomainPollInterval, requestCtx.Duration)
	})
}
