package opencost

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

func Test__OnCostExceedsThreshold__Setup(t *testing.T) {
	trigger := &OnCostExceedsThreshold{}

	t.Run("window is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "", "aggregate": "namespace", "threshold": 100.0},
		})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("aggregate is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "24h", "aggregate": "", "threshold": 100.0},
		})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("threshold must be greater than zero", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace", "threshold": 0.0},
		})
		require.ErrorContains(t, err, "threshold must be greater than zero")
	})

	t.Run("valid setup schedules polling", func(t *testing.T) {
		requestCtx := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace", "threshold": 100.0},
			Requests:      requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, PollAction, requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})
}

func Test__OnCostExceedsThreshold__HandleAction(t *testing.T) {
	trigger := &OnCostExceedsThreshold{}

	t.Run("unknown action returns error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "unknown",
		})
		require.ErrorContains(t, err, "unknown action")
	})

	t.Run("emits event when cost exceeds threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"properties": {"namespace": "production"},
								"window": {"start": "2026-01-19T00:00:00Z", "end": "2026-01-20T00:00:00Z"},
								"start": "2026-01-19T00:00:00Z",
								"end": "2026-01-20T00:00:00Z",
								"minutes": 1440,
								"cpuCost": 80.0,
								"gpuCost": 0,
								"ramCost": 40.0,
								"pvCost": 10.0,
								"networkCost": 5.0,
								"totalCost": 135.0
							}
						}]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace", "threshold": 100.0},
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
		assert.Equal(t, CostExceedsThresholdPayloadType, eventsCtx.Payloads[0].Type)

		payload := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, 135.0, payload["totalCost"])
		assert.Equal(t, 100.0, payload["threshold"])
		assert.Equal(t, "24h", payload["window"])

		assert.Equal(t, PollAction, requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})

	t.Run("does not emit event when cost is below threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"properties": {"namespace": "production"},
								"window": {"start": "2026-01-19T00:00:00Z", "end": "2026-01-20T00:00:00Z"},
								"start": "2026-01-19T00:00:00Z",
								"end": "2026-01-20T00:00:00Z",
								"minutes": 1440,
								"cpuCost": 10.0,
								"gpuCost": 0,
								"ramCost": 5.0,
								"pvCost": 2.0,
								"networkCost": 1.0,
								"totalCost": 18.0
							}
						}]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace", "threshold": 100.0},
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
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})

	t.Run("API error reschedules and returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:          PollAction,
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace", "threshold": 100.0},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: requestCtx,
		})

		require.ErrorContains(t, err, "failed to fetch allocation")
		assert.Len(t, eventsCtx.Payloads, 0)
		assert.Equal(t, PollAction, requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})
}

func Test__parseAndValidateThresholdConfig(t *testing.T) {
	t.Run("sanitizes window and aggregate", func(t *testing.T) {
		config, err := parseAndValidateThresholdConfig(map[string]any{
			"window":    "  24h  ",
			"aggregate": "  NAMESPACE  ",
			"threshold": 100.0,
		})

		require.NoError(t, err)
		assert.Equal(t, "24h", config.Window)
		assert.Equal(t, "namespace", config.Aggregate)
		assert.Equal(t, 100.0, config.Threshold)
	})
}

// Ensure PollInterval is reasonable.
func Test__PollInterval(t *testing.T) {
	assert.Equal(t, 5*time.Minute, PollInterval)
}
