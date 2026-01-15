package dash0

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssueStatus__Setup(t *testing.T) {
	trigger := OnIssueStatus{}

	t.Run("minutesInterval is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{},
		})

		require.ErrorContains(t, err, "minutesInterval is required")
	})

	t.Run("minutesInterval must be between 1 and 59", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"minutesInterval": 0},
		})

		require.ErrorContains(t, err, "minutesInterval must be between 1 and 59")

		err = trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"minutesInterval": 60},
		})

		require.ErrorContains(t, err, "minutesInterval must be between 1 and 59")
	})

	t.Run("successful setup schedules action", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		requestCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
			Requests:        requestCtx,
			Configuration:   map[string]any{"minutesInterval": 5},
		})

		require.NoError(t, err)
		assert.Equal(t, "checkIssues", requestCtx.Action)
		assert.Greater(t, requestCtx.Duration, time.Duration(0))
		assert.Less(t, requestCtx.Duration, 6*time.Minute)
	})

	t.Run("configuration unchanged -> no reschedule", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		requestCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{}

		// First setup
		err := trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
			Requests:        requestCtx,
			Configuration:   map[string]any{"minutesInterval": 5},
		})
		require.NoError(t, err)

		firstDuration := requestCtx.Duration

		// Second setup with same config
		requestCtx2 := &contexts.RequestContext{}
		err = trigger.Setup(core.TriggerContext{
			Logger:          log.NewEntry(log.StandardLogger()),
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
			Requests:        requestCtx2,
			Configuration:   map[string]any{"minutesInterval": 5},
		})
		require.NoError(t, err)

		// Should not schedule if within 1 second
		if requestCtx2.Duration > 0 {
			assert.InDelta(t, firstDuration.Seconds(), requestCtx2.Duration.Seconds(), 1.0)
		}
	})
}

func Test__OnIssueStatus__HandleAction(t *testing.T) {
	trigger := OnIssueStatus{}

	t.Run("unsupported action -> error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:   "unsupported",
			Logger: log.NewEntry(log.StandardLogger()),
		})

		require.ErrorContains(t, err, "action unsupported not supported")
	})

	t.Run("checkIssues with no results -> no event emitted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		eventCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "checkIssues",
			Configuration: map[string]any{
				"minutesInterval": 5,
			},
			Logger:          log.NewEntry(log.StandardLogger()),
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Events:          eventCtx,
			Requests:        requestCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
		// Should still reschedule
		assert.Equal(t, "checkIssues", requestCtx.Action)
	})

	t.Run("checkIssues with results -> event emitted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {"otel_metric_name": "dash0.issue.status"},
										"value": [1234567890, "1"]
									},
									{
										"metric": {"otel_metric_name": "dash0.issue.status"},
										"value": [1234567890, "2"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		eventCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "checkIssues",
			Configuration: map[string]any{
				"minutesInterval": 5,
			},
			Logger:          log.NewEntry(log.StandardLogger()),
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Events:          eventCtx,
			Requests:        requestCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "dash0.issue.detected", eventCtx.Payloads[0].Type)
		payload := eventCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, 2, payload["count"])
		// Should reschedule
		assert.Equal(t, "checkIssues", requestCtx.Action)
	})

	t.Run("query failure -> reschedules anyway", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		eventCtx := &contexts.EventContext{}
		requestCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "checkIssues",
			Configuration: map[string]any{
				"minutesInterval": 5,
			},
			Logger:          log.NewEntry(log.StandardLogger()),
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Events:          eventCtx,
			Requests:        requestCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		// No events emitted on error
		assert.Equal(t, 0, eventCtx.Count())
		// Should still reschedule
		assert.Equal(t, "checkIssues", requestCtx.Action)
	})
}

func Test__OnIssueStatus__Actions(t *testing.T) {
	trigger := OnIssueStatus{}
	actions := trigger.Actions()

	require.Len(t, actions, 1)
	assert.Equal(t, "checkIssues", actions[0].Name)
	assert.False(t, actions[0].UserAccessible)
}
