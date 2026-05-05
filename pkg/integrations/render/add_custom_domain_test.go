package render

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Render_AddCustomDomain__Setup(t *testing.T) {
	component := &AddCustomDomain{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"domain": "app.example.com"},
		})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing domain -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123"},
		})
		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "domain": "app.example.com"},
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
				"service": "srv-123",
				"domain":  "app.example.com",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, RenderServiceNodeMetadata{
			Service: &RenderServiceMetadata{ID: "srv-123", Name: "backend-api"},
		}, metadataCtx.Get())
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
				"domain":              "app.example.com",
				"waitForVerification": false,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
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

	t.Run("array response -> picks added domain and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`[
							{
								"id":"cdm-existing",
								"name":"existing.example.com",
								"verificationStatus":"unverified"
							},
							{
								"id":"cdm-abc123",
								"name":"app.example.com",
								"verificationStatus":"unverified"
							}
						]`,
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
				"domain":              "app.example.com",
				"waitForVerification": false,
			},
		})

		require.NoError(t, err)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "cdm-abc123", data["id"])
		assert.Equal(t, "app.example.com", data["name"])
		assert.Equal(t, "srv-123", data["serviceId"])
	})

	t.Run("missing domain id -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
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
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.ErrorContains(t, err, "custom domain response missing id")
		assert.Empty(t, executionState.KVs)
		assert.Empty(t, metadataCtx.Get())
	})

	t.Run("waitForVerification true, already verified -> emits payload immediately", func(t *testing.T) {
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
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
	})

	t.Run("waitForVerification true, verification failed -> emits payload immediately", func(t *testing.T) {
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
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
	})

	t.Run("waitForVerification true, unverified -> schedules poll without blocking", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}

		started := time.Now()
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service":             "srv-123",
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)
		assert.Less(t, time.Since(started), time.Second, "Execute must not block while waiting for verification")
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, AddCustomDomainPollInterval, requestCtx.Duration)

		require.Len(t, httpCtx.Requests, 3)
		request := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/custom-domains/cdm-abc123/verify")
		request = httpCtx.Requests[2]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/custom-domains/cdm-abc123")

		md, ok := metadataCtx.Get().(AddCustomDomainExecutionMetadata)
		require.True(t, ok)
		require.NotNil(t, md.CustomDomain)
		assert.Equal(t, "unverified", md.CustomDomain.VerificationStatus)
	})

	t.Run("waitForVerification true, retrieved domain is verified -> emits payload without polling", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"verified"}`,
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
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
		assert.Empty(t, requestCtx.Action)
		require.Len(t, httpCtx.Requests, 3)
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/v1/services/srv-123/custom-domains/cdm-abc123/verify")
		assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
	})

	t.Run("waitForVerification true, retrieved domain is failed -> emits payload without polling", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"failed"}`,
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
				"domain":              "app.example.com",
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
		assert.Empty(t, requestCtx.Action)
	})
}

func Test__Render_AddCustomDomain__Poll(t *testing.T) {
	component := &AddCustomDomain{}

	t.Run("missing custom domain id -> error", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{Metadata: AddCustomDomainExecutionMetadata{}},
			Configuration: map[string]any{
				"service": "srv-123",
				"domain":  "app.example.com",
			},
		})

		require.ErrorContains(t, err, "custom domain metadata missing id")
	})

	t.Run("verified -> emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"verified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{Metadata: AddCustomDomainExecutionMetadata{
			CustomDomain: &CustomDomainMetadata{ID: "cdm-abc123", Name: "app.example.com", ServiceID: "srv-123"},
		}}
		requestCtx := &contexts.RequestContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service": "srv-123",
				"domain":  "app.example.com",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Empty(t, requestCtx.Action)
	})

	t.Run("unverified -> reschedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"unverified"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{Metadata: AddCustomDomainExecutionMetadata{
			CustomDomain: &CustomDomainMetadata{ID: "cdm-abc123", Name: "app.example.com", ServiceID: "srv-123"},
		}}
		requestCtx := &contexts.RequestContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service": "srv-123",
				"domain":  "app.example.com",
			},
		})

		require.NoError(t, err)
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, AddCustomDomainPollInterval, requestCtx.Duration)
	})

	t.Run("failed -> emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"cdm-abc123","name":"app.example.com","serviceId":"srv-123","verificationStatus":"failed"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{Metadata: AddCustomDomainExecutionMetadata{
			CustomDomain: &CustomDomainMetadata{ID: "cdm-abc123", Name: "app.example.com", ServiceID: "srv-123"},
		}}
		requestCtx := &contexts.RequestContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service": "srv-123",
				"domain":  "app.example.com",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Empty(t, requestCtx.Action)

		require.Len(t, executionState.Payloads, 1)
		emitted := readMap(executionState.Payloads[0])
		data := readMap(emitted["data"])
		assert.Equal(t, "failed", data["verificationStatus"])
	})
}
