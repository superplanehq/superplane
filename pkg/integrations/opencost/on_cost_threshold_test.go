package opencost

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

func Test__OnCostThreshold__Setup(t *testing.T) {
	trigger := &OnCostThreshold{}

	t.Run("window is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "", "aggregate": "namespace", "threshold": "100"},
		})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("aggregate is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "1d", "aggregate": "", "threshold": "100"},
		})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("valid setup schedules polling", func(t *testing.T) {
		requestCtx := &contexts.RequestContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace", "threshold": "100"},
			Requests:      requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, PollAction, requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})
}

func Test__OnCostThreshold__HandleAction(t *testing.T) {
	trigger := &OnCostThreshold{}

	t.Run("unknown action returns error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "unknown",
		})
		require.ErrorContains(t, err, "unknown action")
	})
}

func Test__OnCostThreshold__Poll(t *testing.T) {
	trigger := &OnCostThreshold{}

	t.Run("emits events for costs exceeding threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [
							{
								"production": {
									"name": "production",
									"properties": {"namespace": "production"},
									"start": "2026-02-21T00:00:00Z",
									"end": "2026-02-22T00:00:00Z",
									"cpuCost": 45.23,
									"gpuCost": 0,
									"ramCost": 38.92,
									"pvCost": 8.75,
									"networkCost": 12.5,
									"totalCost": 105.4,
									"cpuEfficiency": 0.42,
									"ramEfficiency": 0.61,
									"totalEfficiency": 0.51
								},
								"kube-system": {
									"name": "kube-system",
									"properties": {"namespace": "kube-system"},
									"start": "2026-02-21T00:00:00Z",
									"end": "2026-02-22T00:00:00Z",
									"cpuCost": 12.1,
									"gpuCost": 0,
									"ramCost": 8.45,
									"pvCost": 0,
									"networkCost": 3.2,
									"totalCost": 23.75,
									"cpuEfficiency": 0.35,
									"ramEfficiency": 0.28,
									"totalEfficiency": 0.31
								}
							}
						]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace", "threshold": "100"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: requestCtx,
		})

		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
		assert.Equal(t, CostThresholdPayloadType, eventsCtx.Payloads[0].Type)
		payload := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "production", payload["name"])
		assert.Equal(t, 105.4, payload["totalCost"])
		assert.Equal(t, float64(100), payload["threshold"])

		assert.Equal(t, PollAction, requestCtx.Action)
	})

	t.Run("no events emitted when all costs below threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [
							{
								"kube-system": {
									"name": "kube-system",
									"totalCost": 23.75
								}
							}
						]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace", "threshold": "100"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: requestCtx,
		})

		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
		assert.Equal(t, PollAction, requestCtx.Action)
	})

	t.Run("API error reschedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`error`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace", "threshold": "100"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: requestCtx,
		})

		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
		assert.Equal(t, PollAction, requestCtx.Action)
	})
}

func Test__parseAndValidateOnCostThresholdConfiguration(t *testing.T) {
	t.Run("valid configuration with string threshold", func(t *testing.T) {
		config, err := parseAndValidateOnCostThresholdConfiguration(map[string]any{
			"window":    "1d",
			"aggregate": "namespace",
			"threshold": "100.50",
		})

		require.NoError(t, err)
		assert.Equal(t, "1d", config.Window)
		assert.Equal(t, "namespace", config.Aggregate)
		assert.Equal(t, 100.50, config.Threshold)
	})

	t.Run("sanitizes whitespace", func(t *testing.T) {
		config, err := parseAndValidateOnCostThresholdConfiguration(map[string]any{
			"window":    "  7d  ",
			"aggregate": "  CLUSTER  ",
			"threshold": "50",
		})

		require.NoError(t, err)
		assert.Equal(t, "7d", config.Window)
		assert.Equal(t, "cluster", config.Aggregate)
	})

	t.Run("negative threshold returns error", func(t *testing.T) {
		_, err := parseAndValidateOnCostThresholdConfiguration(map[string]any{
			"window":    "1d",
			"aggregate": "namespace",
			"threshold": -10.0,
		})

		require.ErrorContains(t, err, "threshold must be a non-negative number")
	})

	t.Run("invalid threshold string returns error", func(t *testing.T) {
		_, err := parseAndValidateOnCostThresholdConfiguration(map[string]any{
			"window":    "1d",
			"aggregate": "namespace",
			"threshold": "not-a-number",
		})

		require.Error(t, err)
	})
}
