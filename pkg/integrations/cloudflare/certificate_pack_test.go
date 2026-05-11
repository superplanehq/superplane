package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OrderCertificatePack__Setup(t *testing.T) {
	component := &OrderCertificatePack{}

	t.Run("missing zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"hosts":                []any{"example.com"},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing hosts returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
		})
		require.ErrorContains(t, err, "at least one host is required")
	})

	t.Run("blank host in list returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{""},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
		})
		require.ErrorContains(t, err, "must not be blank")
	})

	t.Run("missing certificateAuthority returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"hosts":            []any{"example.com"},
				"validationMethod": "txt",
			},
		})
		require.ErrorContains(t, err, "certificateAuthority is required")
	})

	t.Run("missing validationMethod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "lets_encrypt",
			},
		})
		require.ErrorContains(t, err, "validationMethod is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com", "*.example.com"},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
		})
		require.NoError(t, err)
	})
}

func Test__OrderCertificatePack__Execute(t *testing.T) {
	component := &OrderCertificatePack{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "token123"},
		Metadata:      Metadata{Zones: []Zone{{ID: "zone123", Name: "example.com"}}},
	}

	t.Run("orders certificate pack and emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "pack-abc123",
							"certificate_authority": "lets_encrypt",
							"hosts": ["preview.example.com"],
							"status": "initializing",
							"type": "advanced",
							"validation_method": "txt"
						}
					}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"preview.example.com"},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, OrderCertificatePackPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone123/ssl/certificate_packs/order",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Equal(t, "lets_encrypt", body["certificate_authority"])
		assert.Equal(t, "advanced", body["type"])
		assert.Equal(t, "txt", body["validation_method"])
		hosts, ok := body["hosts"].([]any)
		require.True(t, ok)
		assert.Equal(t, "preview.example.com", hosts[0])
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"message":"Invalid host"}]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"bad_host"},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "txt",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to order certificate pack")
	})
}

func Test__DeleteCertificatePack__Setup(t *testing.T) {
	component := &DeleteCertificatePack{}

	t.Run("missing certificatePack returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.ErrorContains(t, err, "certificatePack is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"certificatePack": "zone123/pack-abc",
			},
		})
		require.NoError(t, err)
	})
}

func Test__DeleteCertificatePack__Execute(t *testing.T) {
	component := &DeleteCertificatePack{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "token123"},
	}

	t.Run("deletes certificate pack and emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "zone123/pack-abc",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, DeleteCertificatePackPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone123/ssl/certificate_packs/pack-abc",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "zone123", payload["zoneId"])
		assert.Equal(t, "pack-abc", payload["packId"])
		assert.Equal(t, true, payload["deleted"])
	})

	t.Run("parses zone and pack from slash-separated value", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`))},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "abczone/defpack",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/abczone/ssl/certificate_packs/defpack",
			httpContext.Requests[0].URL.String(),
		)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"message":"Certificate pack not found"}]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "zone123/pack-abc",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete certificate pack")
	})
}
