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

func Test__OnCostExceedsThreshold__Setup(t *testing.T) {
	trigger := &OnCostExceedsThreshold{}

	t.Run("window is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "",
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
		})

		require.ErrorContains(t, err, "window is required")
	})

	t.Run("invalid window returns error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "30d",
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
		})

		require.ErrorContains(t, err, "invalid window")
	})

	t.Run("aggregate is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": "",
				"threshold": 50.0,
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
		})

		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("threshold must be greater than zero", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"threshold": 0.0,
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
		})

		require.ErrorContains(t, err, "threshold must be greater than zero")
	})

	t.Run("valid setup schedules poll action", func(t *testing.T) {
		requestCtx := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, DefaultPollInterval, requestCtx.Duration)
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

	t.Run("poll emits events for allocations exceeding threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 30.0,
								"gpuCost": 0,
								"ramCost": 20.0,
								"pvCost": 5.0,
								"networkCost": 2.0,
								"totalCost": 57.0
							},
							"staging": {
								"name": "staging",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 5.0,
								"gpuCost": 0,
								"ramCost": 3.0,
								"pvCost": 1.0,
								"networkCost": 0.5,
								"totalCost": 9.5
							}
						}]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
		assert.Equal(t, CostAllocationPayloadType, eventsCtx.Payloads[0].Type)
		payload := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "production", payload["name"])
		assert.Equal(t, 57.0, payload["totalCost"])
		assert.Equal(t, 50.0, payload["threshold"])
	})

	t.Run("poll with filter only emits matching allocations", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 30.0,
								"gpuCost": 0,
								"ramCost": 20.0,
								"pvCost": 5.0,
								"networkCost": 2.0,
								"totalCost": 57.0
							},
							"staging": {
								"name": "staging",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 30.0,
								"gpuCost": 0,
								"ramCost": 20.0,
								"pvCost": 5.0,
								"networkCost": 2.0,
								"totalCost": 57.0
							}
						}]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
				"filter":    "staging",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
		payload := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "staging", payload["name"])
	})

	t.Run("poll does not emit when under threshold", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"staging": {
								"name": "staging",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 5.0,
								"gpuCost": 0,
								"ramCost": 3.0,
								"pvCost": 1.0,
								"networkCost": 0.5,
								"totalCost": 9.5
							}
						}]
					}`)),
				},
			},
		}

		eventsCtx := &contexts.EventContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"threshold": 50.0,
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			Events:   eventsCtx,
			Requests: &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
	})
}
