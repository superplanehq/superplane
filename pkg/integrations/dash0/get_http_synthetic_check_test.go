package dash0

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

func Test__GetHTTPSyntheticCheck__Setup(t *testing.T) {
	component := GetHTTPSyntheticCheck{}

	t.Run("checkId is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "checkId is required")
	})

	t.Run("checkId cannot be empty", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkId": ""},
		})

		require.ErrorContains(t, err, "checkId is required")
	})

	t.Run("checkId cannot be whitespace", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkId": "   "},
		})

		require.ErrorContains(t, err, "checkId is required")
	})

	t.Run("dataset is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkId": "64617368-3073-796e-7468-73599f287bf4"},
		})

		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("valid setup skips API when metadata already set", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"checkName": "Already Set"},
			},
			Configuration: map[string]any{
				"checkId": "64617368-3073-796e-7468-73599f287bf4",
				"dataset": "default",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup fetches check name from API", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							{
								"kind": "Dash0SyntheticCheck",
								"metadata": {"name": "My Health Check"},
								"spec": {
									"display": {"name": "My Health Check"},
									"enabled": true,
									"labels": {},
									"notifications": {"channels": [], "onlyCriticalChannels": []},
									"plugin": {"kind": "http", "spec": {"assertions": {}, "request": {"url": "https://example.com", "method": "get"}}},
									"retries": {"kind": "off", "spec": {}},
									"schedule": {"interval": "1m", "locations": ["us-east-1"], "strategy": "all_locations"}
								}
							}
						`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"checkId": "64617368-3073-796e-7468-73599f287bf4",
				"dataset": "production",
			},
		})

		require.NoError(t, err)
		storedMetadata := metadata.Metadata.(GetHTTPSyntheticCheckNodeMetadata)
		assert.Equal(t, "My Health Check", storedMetadata.CheckName)
	})
}

func Test__GetHTTPSyntheticCheck__Execute(t *testing.T) {
	component := GetHTTPSyntheticCheck{}

	t.Run("successful fetch with all metrics", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSyntheticCheck response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"kind": "Dash0SyntheticCheck",
							"metadata": {
								"annotations": {},
								"description": "",
								"labels": {
									"dash0.com/dataset": "default",
									"dash0.com/id": "64617368-3073-796e-7468-73599f287bf4",
									"dash0.com/origin": "",
									"dash0.com/version": "21"
								},
								"name": "New synthetic check"
							},
							"spec": {
								"display": {
									"name": "New synthetic check"
								},
								"enabled": true,
								"labels": {},
								"notifications": {
									"channels": [],
									"onlyCriticalChannels": []
								},
								"plugin": {
									"kind": "http",
									"spec": {
										"assertions": {
											"criticalAssertions": [
												{
													"kind": "status_code",
													"spec": {
														"operator": "is",
														"value": "200"
													}
												}
											],
											"degradedAssertions": []
										},
										"request": {
											"headers": [],
											"method": "get",
											"queryParameters": [],
											"redirects": "follow",
											"tls": {
												"allowInsecure": false
											},
											"tracing": {
												"addTracingHeaders": true
											},
											"url": "https://example.com/health"
										}
									}
								},
								"retries": {
									"kind": "off",
									"spec": {}
								},
								"schedule": {
									"interval": "1m",
									"locations": ["be-brussels"],
									"strategy": "all_locations"
								}
							}
						}
					`)),
				},
				// totalRuns24h
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "61"]
									}
								]
							}
						}
					`)),
				},
				// healthyRuns24h
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "58"]
									}
								]
							}
						}
					`)),
				},
				// criticalRuns24h
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "3"]
									}
								]
							}
						}
					`)),
				},
				// totalDuration24h
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "32.94"]
									}
								]
							}
						}
					`)),
				},
				// totalRuns7d
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "402"]
									}
								]
							}
						}
					`)),
				},
				// healthyRuns7d
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "390"]
									}
								]
							}
						}
					`)),
				},
				// criticalRuns7d
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "12"]
									}
								]
							}
						}
					`)),
				},
				// totalDuration7d
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {},
										"value": [1234567890, "209.04"]
									}
								]
							}
						}
					`)),
				},
				// lastOutcome
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {"dash0_synthetic_check_outcome": "Healthy"},
										"value": [1234567890, "1234567890"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "64617368-3073-796e-7468-73599f287bf4",
				"dataset": "default",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameHealthy, execCtx.Channel)
		assert.Equal(t, "dash0.syntheticCheck.fetched", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		// Payloads are wrapped in a structure with type, timestamp, and data
		wrappedPayload := execCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.NotNil(t, payload["configuration"])
		assert.NotNil(t, payload["metrics"])

		config := payload["configuration"].(*SyntheticCheckResponse)
		assert.Equal(t, "Dash0SyntheticCheck", config.Kind)
		assert.Equal(t, "New synthetic check", config.Metadata.Name)
		assert.Equal(t, "https://example.com/health", config.Spec.Plugin.Spec.Request.URL)

		metrics := payload["metrics"].(*SyntheticCheckMetrics)
		assert.Equal(t, 61, metrics.TotalRuns24h)
		assert.Equal(t, 58, metrics.HealthyRuns24h)
		assert.Equal(t, 3, metrics.CriticalRuns24h)
		assert.InDelta(t, 540.0, metrics.AvgDuration24h, 1.0)

		assert.Equal(t, 402, metrics.TotalRuns7d)
		assert.Equal(t, 390, metrics.HealthyRuns7d)
		assert.Equal(t, 12, metrics.CriticalRuns7d)
		assert.InDelta(t, 520.0, metrics.AvgDuration7d, 1.0)
	})

	t.Run("successful fetch with partial metrics", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSyntheticCheck response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"kind": "Dash0SyntheticCheck",
							"metadata": {
								"annotations": {},
								"description": "",
								"labels": {
									"dash0.com/dataset": "default",
									"dash0.com/id": "test-check-id"
								},
								"name": "Test Check"
							},
							"spec": {
								"display": {"name": "Test Check"},
								"enabled": true,
								"labels": {},
								"notifications": {"channels": [], "onlyCriticalChannels": []},
								"plugin": {
									"kind": "http",
									"spec": {
										"assertions": {"criticalAssertions": [], "degradedAssertions": []},
										"request": {"url": "https://test.com", "method": "get"}
									}
								},
								"retries": {"kind": "off", "spec": {}},
								"schedule": {"interval": "1m", "locations": ["us-east-1"], "strategy": "all_locations"}
							}
						}
					`)),
				},
				// totalRuns24h - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// healthyRuns24h - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// criticalRuns24h - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// totalDuration24h - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// totalRuns7d - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// healthyRuns7d - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// criticalRuns7d - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// totalDuration7d - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				// lastOutcome - returns empty result
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "test-check-id",
				"dataset": "default",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)

		// Empty outcome means Pass() is called — no payloads emitted
		assert.Empty(t, execCtx.Payloads)
		assert.Empty(t, execCtx.Channel)
	})

	t.Run("uses default dataset when not provided", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSyntheticCheck response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"kind": "Dash0SyntheticCheck",
							"metadata": {
								"annotations": {},
								"description": "",
								"labels": {"dash0.com/dataset": "default"},
								"name": "Test"
							},
							"spec": {
								"display": {"name": "Test"},
								"enabled": true,
								"labels": {},
								"notifications": {"channels": [], "onlyCriticalChannels": []},
								"plugin": {
									"kind": "http",
									"spec": {
										"assertions": {"criticalAssertions": [], "degradedAssertions": []},
										"request": {"url": "https://test.com", "method": "get"}
									}
								},
								"retries": {"kind": "off", "spec": {}},
								"schedule": {"interval": "1m", "locations": ["us-east-1"], "strategy": "all_locations"}
							}
						}
					`)),
				},
				// Metrics responses (8 scalar + 1 outcome = 9 total)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "test-check-id",
				// dataset not provided
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
	})

	t.Run("handles API error when fetching check", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "check not found"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "non-existent-id",
				"dataset": "default",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get synthetic check")
	})

	t.Run("handles missing integration configuration", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "test-id",
				"dataset": "default",
			},
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error creating client")
	})

	t.Run("emits on critical channel when outcome is Critical", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSyntheticCheck response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"kind": "Dash0SyntheticCheck",
							"metadata": {"name": "Failing Check"},
							"spec": {
								"display": {"name": "Failing Check"},
								"enabled": true,
								"labels": {},
								"notifications": {"channels": [], "onlyCriticalChannels": []},
								"plugin": {
									"kind": "http",
									"spec": {
										"assertions": {"criticalAssertions": [], "degradedAssertions": []},
										"request": {"url": "https://down.example.com", "method": "get"}
									}
								},
								"retries": {"kind": "off", "spec": {}},
								"schedule": {"interval": "1m", "locations": ["us-east-1"], "strategy": "all_locations"}
							}
						}
					`)),
				},
				// 8 scalar metric responses (empty)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))},
				// lastOutcome - Critical
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {"dash0_synthetic_check_outcome": "Critical"},
										"value": [1234567890, "1234567890"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "failing-check-id",
				"dataset": "default",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameCritical, execCtx.Channel)
		assert.Equal(t, "dash0.syntheticCheck.fetched", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})
}
