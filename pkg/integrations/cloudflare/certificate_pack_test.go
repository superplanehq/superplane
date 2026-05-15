package cloudflare

import (
	"encoding/json"
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

func Test__OrderCertificatePack__Setup(t *testing.T) {
	component := &OrderCertificatePack{}

	t.Run("configuration excludes deprecated DigiCert and conditionally requires validity days", func(t *testing.T) {
		fields := component.Configuration()
		var caField *configuration.Field
		var validationMethodField *configuration.Field
		var validityField *configuration.Field
		for i := range fields {
			switch fields[i].Name {
			case "certificateAuthority":
				caField = &fields[i]
			case "validationMethod":
				validationMethodField = &fields[i]
			case "validityDays":
				validityField = &fields[i]
			}
		}

		require.NotNil(t, caField)
		require.NotNil(t, caField.TypeOptions)
		require.NotNil(t, caField.TypeOptions.Select)
		for _, option := range caField.TypeOptions.Select.Options {
			assert.NotEqual(t, "digicert", option.Value)
		}

		require.NotNil(t, validationMethodField)
		require.NotNil(t, validationMethodField.TypeOptions)
		require.NotNil(t, validationMethodField.TypeOptions.Select)
		for _, option := range validationMethodField.TypeOptions.Select.Options {
			assert.NotEqual(t, "cname", option.Value)
		}

		require.NotNil(t, validityField)
		assert.Equal(t, "90", validityField.Default)
		assert.Equal(t, []configuration.VisibilityCondition{
			{Field: "certificateAuthority", Values: []string{"google", "ssl_com"}},
		}, validityField.VisibilityConditions)
		assert.Equal(t, []configuration.RequiredCondition{
			{Field: "certificateAuthority", Values: []string{"google", "ssl_com"}},
		}, validityField.RequiredConditions)
	})

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

	t.Run("deprecated DigiCert returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "digicert",
				"validationMethod":     "txt",
			},
		})
		require.ErrorContains(t, err, "unsupported certificateAuthority")
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

	t.Run("unsupported validationMethod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "lets_encrypt",
				"validationMethod":     "cname",
			},
		})
		require.ErrorContains(t, err, "unsupported validationMethod")
	})

	t.Run("missing validityDays for SSL.com returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "ssl_com",
				"validationMethod":     "txt",
			},
		})
		require.ErrorContains(t, err, "validityDays is required")
	})

	t.Run("invalid validityDays returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "google",
				"validationMethod":     "txt",
				"validityDays":         "45",
			},
		})
		require.ErrorContains(t, err, "validityDays must be one of")
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

	t.Run("valid Google configuration passes with validityDays", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com", "*.example.com"},
				"certificateAuthority": "google",
				"validationMethod":     "txt",
				"validityDays":         "90",
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

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "example.com", payload["zoneName"])
	})

	t.Run("orders Google certificate pack with validity days", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "pack-google",
							"certificate_authority": "google",
							"hosts": ["example.com"],
							"status": "initializing",
							"type": "advanced",
							"validation_method": "txt",
							"validity_days": 90
						}
					}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":                 "zone123",
				"hosts":                []any{"example.com"},
				"certificateAuthority": "google",
				"validationMethod":     "txt",
				"validityDays":         "90",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.NoError(t, err)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Equal(t, float64(90), body["validity_days"])
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

	t.Run("configuration stores selected certificate pack by readable name", func(t *testing.T) {
		fields := component.Configuration()
		require.Len(t, fields, 1)
		require.NotNil(t, fields[0].TypeOptions)
		require.NotNil(t, fields[0].TypeOptions.Resource)
		assert.Equal(t, "certificate_pack", fields[0].TypeOptions.Resource.Type)
		assert.True(t, fields[0].TypeOptions.Resource.UseNameAsValue)
	})

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
		Metadata: Metadata{
			Zones: []Zone{{ID: "zone123", Name: "zone.example.com"}},
		},
	}

	t.Run("deletes certificate pack and emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`)),
				},
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

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone123/ssl/certificate_packs",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone123/ssl/certificate_packs/pack-abc",
			httpContext.Requests[1].URL.String(),
		)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "zone123", payload["zoneId"])
		assert.Equal(t, "pack-abc", payload["packId"])
		assert.Equal(t, true, payload["deleted"])
		assert.Equal(t, "zone.example.com", payload["zoneName"])
		_, hasHosts := payload["hosts"]
		assert.False(t, hasHosts)
	})

	t.Run("parses zone and pack from slash-separated value", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`))},
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
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/abczone/ssl/certificate_packs",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/abczone/ssl/certificate_packs/defpack",
			httpContext.Requests[1].URL.String(),
		)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`))},
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

	t.Run("resolves zone from metadata when certificate pack id has no slash", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": [{"id": "found-pack-id", "hosts": ["preview.example.com"]}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationZones := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token123"},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "z-first", Name: "first.example.com"},
					{ID: "z-second", Name: "second.example.com"},
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "found-pack-id",
			},
			HTTP:           httpContext,
			Integration:    integrationZones,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/z-first/ssl/certificate_packs",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/z-second/ssl/certificate_packs",
			httpContext.Requests[1].URL.String(),
		)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodGet, httpContext.Requests[1].Method)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/z-second/ssl/certificate_packs/found-pack-id",
			httpContext.Requests[2].URL.String(),
		)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[2].Method)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "z-second", payload["zoneId"])
		assert.Equal(t, "found-pack-id", payload["packId"])
		assert.Equal(t, "second.example.com", payload["zoneName"])
		assert.Equal(t, []string{"preview.example.com"}, payload["hosts"])
	})

	t.Run("resolves readable resource name to certificate pack id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": [{"id": "found-pack-id", "hosts": ["preview.example.com"]}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationZones := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token123"},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "z-first", Name: "first.example.com"},
					{ID: "z-second", Name: "second.example.com"},
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "second.example.com - preview.example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationZones,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/z-second/ssl/certificate_packs/found-pack-id",
			httpContext.Requests[2].URL.String(),
		)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[2].Method)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "z-second", payload["zoneId"])
		assert.Equal(t, "found-pack-id", payload["packId"])
		assert.Equal(t, "second.example.com", payload["zoneName"])
		assert.Equal(t, []string{"preview.example.com"}, payload["hosts"])
	})

	t.Run("pack id only returns error when pack is not in any configured zone", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":[]}`))},
			},
		}
		integrationZones := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token123"},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "z1", Name: "a.example.com"},
					{ID: "z2", Name: "b.example.com"},
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "missing-pack-id",
			},
			HTTP:           httpContext,
			Integration:    integrationZones,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in any configured zone")
		require.Len(t, httpContext.Requests, 2)
	})

	t.Run("slash form resolves zone name via integration metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": [{"id": "pack-xyz", "hosts": ["svc.named.example.com"]}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationZones := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token123"},
			Metadata: Metadata{
				Zones: []Zone{{ID: "zone-by-id", Name: "named.example.com"}},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"certificatePack": "named.example.com/pack-xyz",
			},
			HTTP:           httpContext,
			Integration:    integrationZones,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone-by-id/ssl/certificate_packs",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/zones/zone-by-id/ssl/certificate_packs/pack-xyz",
			httpContext.Requests[1].URL.String(),
		)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "named.example.com", payload["zoneName"])
		assert.Equal(t, []string{"svc.named.example.com"}, payload["hosts"])
	})
}
