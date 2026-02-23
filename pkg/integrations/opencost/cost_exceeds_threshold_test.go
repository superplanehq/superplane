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

func Test__CostExceedsThreshold__Setup(t *testing.T) {
	trigger := &CostExceedsThreshold{}

	t.Run("missing window -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"aggregate": "namespace",
				"threshold": 100.0,
			},
		})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("missing aggregate -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "1d",
				"threshold": 100.0,
			},
		})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("zero threshold -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
				"threshold": 0.0,
			},
		})
		require.ErrorContains(t, err, "threshold must be greater than 0")
	})

	t.Run("negative threshold -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
				"threshold": -10.0,
			},
		})
		require.ErrorContains(t, err, "threshold must be greater than 0")
	})

	t.Run("valid configuration -> schedules polling", func(t *testing.T) {
		requestCtx := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
				"threshold": 100.0,
			},
			Requests: requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, CostExceedsThresholdPollAction, requestCtx.Action)
		assert.Equal(t, CostExceedsThresholdPollInterval, requestCtx.Duration)
	})
}

func Test__CostExceedsThreshold__HandleAction(t *testing.T) {
	trigger := &CostExceedsThreshold{}

	t.Run("unknown action -> error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "unknown",
		})
		require.ErrorContains(t, err, "unknown action")
	})

	t.Run("cost above threshold -> emits event", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"status": "success",
						"data": [{
							"default": {
								"name": "default",
								"start": "2026-02-22T00:00:00Z",
								"end": "2026-02-23T00:00:00Z",
								"cpuCost": 85.5,
								"gpuCost": 0,
								"ramCost": 45.25,
								"pvCost": 12.5,
								"networkCost": 7.5,
								"totalCost": 150.75,
								"totalEfficiency": 0.6
							}
						}]
					}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: CostExceedsThresholdPollAction,
			Configuration: CostExceedsThresholdConfiguration{
				Window:    "1d",
				Aggregate: "namespace",
				Threshold: 100.0,
			},
			HTTP:     httpCtx,
			Events:   events,
			Requests: requests,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiURL": "http://opencost:9003",
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, CostExceedsThresholdPayloadType, events.Payloads[0].Type)

		eventData, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "default", eventData["name"])
		assert.Equal(t, 150.75, eventData["totalCost"])
		assert.Equal(t, 100.0, eventData["threshold"])

		assert.Equal(t, CostExceedsThresholdPollAction, requests.Action)
		assert.Equal(t, CostExceedsThresholdPollInterval, requests.Duration)
	})

	t.Run("cost below threshold -> no event emitted", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"status": "success",
						"data": [{
							"default": {
								"name": "default",
								"start": "2026-02-22T00:00:00Z",
								"end": "2026-02-23T00:00:00Z",
								"cpuCost": 5.0,
								"ramCost": 3.0,
								"totalCost": 8.0,
								"totalEfficiency": 0.5
							}
						}]
					}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: CostExceedsThresholdPollAction,
			Configuration: CostExceedsThresholdConfiguration{
				Window:    "1d",
				Aggregate: "namespace",
				Threshold: 100.0,
			},
			HTTP:     httpCtx,
			Events:   events,
			Requests: requests,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiURL": "http://opencost:9003",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Equal(t, CostExceedsThresholdPollAction, requests.Action)
	})

	t.Run("API error -> continues polling", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{"error":"temporary outage"}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: CostExceedsThresholdPollAction,
			Configuration: CostExceedsThresholdConfiguration{
				Window:    "1d",
				Aggregate: "namespace",
				Threshold: 100.0,
			},
			HTTP:     httpCtx,
			Events:   events,
			Requests: requests,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiURL": "http://opencost:9003",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Equal(t, CostExceedsThresholdPollAction, requests.Action)
		assert.Equal(t, CostExceedsThresholdPollInterval, requests.Duration)
	})

	t.Run("multiple items some above threshold -> emits only exceeding items", func(t *testing.T) {
		events := &contexts.EventContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"status": "success",
						"data": [{
							"kube-system": {
								"name": "kube-system",
								"totalCost": 2.0
							},
							"production": {
								"name": "production",
								"totalCost": 250.0,
								"cpuCost": 150.0,
								"ramCost": 100.0
							}
						}]
					}`)),
				},
			},
		}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: CostExceedsThresholdPollAction,
			Configuration: CostExceedsThresholdConfiguration{
				Window:    "1d",
				Aggregate: "namespace",
				Threshold: 100.0,
			},
			HTTP:     httpCtx,
			Events:   events,
			Requests: requests,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiURL": "http://opencost:9003",
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, 1, events.Count())

		eventData, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "production", eventData["name"])
		assert.Equal(t, 250.0, eventData["totalCost"])
	})
}
