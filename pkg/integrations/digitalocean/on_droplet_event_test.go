package digitalocean

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

func Test__OnDropletEvent__Setup(t *testing.T) {
	trigger := &OnDropletEvent{}

	t.Run("valid config -> stores metadata and schedules poll", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"events": []string{"create", "destroy"},
			},
			Metadata:    metadataCtx,
			Requests:    requestCtx,
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		require.NotNil(t, metadataCtx.Metadata)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 60*time.Second, requestCtx.Duration)
	})

	t.Run("empty events -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"events": []string{},
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "at least one event type must be selected")
	})

	t.Run("metadata already set -> skips setup", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: OnDropletEventMetadata{
				LastPollTime: time.Now().UTC().Format(time.RFC3339),
			},
		}
		requestCtx := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"events": []string{"create"},
			},
			Metadata:    metadataCtx,
			Requests:    requestCtx,
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, requestCtx.Action)
	})
}

func Test__OnDropletEvent__HandleAction(t *testing.T) {
	trigger := &OnDropletEvent{}

	t.Run("poll with new matching actions -> emits events and re-schedules", func(t *testing.T) {
		pastTime := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
		futureTime := time.Now().UTC().Add(1 * time.Minute).Format(time.RFC3339)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"actions": [
							{
								"id": 111,
								"status": "completed",
								"type": "create",
								"started_at": "` + pastTime + `",
								"completed_at": "` + futureTime + `",
								"resource_id": 222,
								"resource_type": "droplet",
								"region_slug": "nyc3"
							},
							{
								"id": 333,
								"status": "completed",
								"type": "destroy",
								"started_at": "` + pastTime + `",
								"completed_at": "` + futureTime + `",
								"resource_id": 444,
								"resource_type": "droplet",
								"region_slug": "sfo3"
							},
							{
								"id": 555,
								"status": "completed",
								"type": "power_on",
								"started_at": "` + pastTime + `",
								"completed_at": "` + futureTime + `",
								"resource_id": 666,
								"resource_type": "droplet",
								"region_slug": "ams3"
							},
							{
								"id": 777,
								"status": "completed",
								"type": "create",
								"started_at": "` + pastTime + `",
								"completed_at": "` + futureTime + `",
								"resource_id": 888,
								"resource_type": "floating_ip",
								"region_slug": "nyc3"
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		eventCtx := &contexts.EventContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: OnDropletEventMetadata{
				LastPollTime: pastTime,
			},
		}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"events": []string{"create", "destroy"},
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Events:      eventCtx,
			Metadata:    metadataCtx,
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, eventCtx.Count())
		assert.Equal(t, "digitalocean.droplet.create", eventCtx.Payloads[0].Type)
		assert.Equal(t, "digitalocean.droplet.destroy", eventCtx.Payloads[1].Type)

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 60*time.Second, requestCtx.Duration)
	})

	t.Run("poll with no new actions -> re-schedules only", func(t *testing.T) {
		now := time.Now().UTC().Format(time.RFC3339)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"actions": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		eventCtx := &contexts.EventContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: OnDropletEventMetadata{
				LastPollTime: now,
			},
		}
		requestCtx := &contexts.RequestContext{}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"events": []string{"create"},
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Events:      eventCtx,
			Metadata:    metadataCtx,
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 60*time.Second, requestCtx.Duration)
	})
}
