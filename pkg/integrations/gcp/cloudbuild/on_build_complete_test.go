package cloudbuild

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnBuildCompleteSetup(t *testing.T) {
	trigger := &OnBuildComplete{}

	t.Run("creates subscription on first setup", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{}
		metadataCtx := &testcontexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, gcpcommon.ActionNameEnsureCloudBuild, integrationCtx.ActionRequests[0].ActionName)

		metadata := OnBuildCompleteMetadata{}
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
		assert.NotEmpty(t, metadata.SubscriptionID)
	})

	t.Run("ensures subscription exists even when metadata already has a subscription id", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{
			Metadata: gcpcommon.Metadata{
				ProjectID:              "demo-project",
				CloudBuildSubscription: "sp-cb-sub-existing",
			},
		}
		metadataCtx := &testcontexts.MetadataContext{
			Metadata: OnBuildCompleteMetadata{SubscriptionID: "existing-id"},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, gcpcommon.ActionNameEnsureCloudBuild, integrationCtx.ActionRequests[0].ActionName)

		metadata := OnBuildCompleteMetadata{}
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
		assert.NotEmpty(t, metadata.SubscriptionID)
	})

	t.Run("returns error when integration is nil", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Metadata: &testcontexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "connect the GCP integration")
	})

	t.Run("rejects direct build source with trigger id filter", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"buildSource": buildSourceDirect,
				"triggerId":   "trigger-abc",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "triggerId cannot be combined")
	})
}

func Test_OnBuildComplete_Metadata(t *testing.T) {
	trigger := &OnBuildComplete{}
	assert.Equal(t, "gcp.cloudbuild.onBuildComplete", trigger.Name())
	assert.Equal(t, "Cloud Build • On Build Complete", trigger.Label())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
	assert.Nil(t, trigger.Actions())
}

func Test_OnBuildComplete_ExampleData(t *testing.T) {
	trigger := &OnBuildComplete{}
	data := trigger.ExampleData()
	assert.NotEmpty(t, data["type"])
	assert.NotEmpty(t, data["timestamp"])
	payload, ok := data["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "SUCCESS", payload["status"])
	assert.NotEmpty(t, payload["id"])
	assert.NotEmpty(t, payload["logUrl"])
}

func Test_OnBuildComplete_OnIntegrationMessage(t *testing.T) {
	trigger := &OnBuildComplete{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("no status filter emits for any terminal status", func(t *testing.T) {
		for _, status := range []string{"SUCCESS", "FAILURE", "CANCELLED", "TIMEOUT"} {
			events := &testcontexts.EventContext{}
			err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
				Configuration: map[string]any{},
				Message: map[string]any{
					"id":     "build-123",
					"status": status,
				},
				Logger: logger,
				Events: events,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, events.Count(), "expected event for status %q", status)
		}
	})

	t.Run("no status filter skips non-terminal statuses", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"id":     "build-123",
				"status": "WORKING",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("status filter matches emits event", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"statuses": []string{"SUCCESS"},
			},
			Message: map[string]any{
				"id":     "build-123",
				"status": "SUCCESS",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("status filter does not match skips event", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"statuses": []string{"SUCCESS"},
			},
			Message: map[string]any{
				"id":     "build-456",
				"status": "FAILURE",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("no build source filter listens to both triggered and direct builds", func(t *testing.T) {
		for _, message := range []map[string]any{
			{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-abc",
			},
			{
				"id":     "build-456",
				"status": "SUCCESS",
			},
		} {
			events := &testcontexts.EventContext{}
			err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
				Configuration: map[string]any{},
				Message:       message,
				Logger:        logger,
				Events:        events,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, events.Count())
		}
	})

	t.Run("triggered build source emits only builds with buildTriggerId", func(t *testing.T) {
		triggeredEvents := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"buildSource": buildSourceTriggered,
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-abc",
			},
			Logger: logger,
			Events: triggeredEvents,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, triggeredEvents.Count())

		directEvents := &testcontexts.EventContext{}
		err = trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"buildSource": buildSourceTriggered,
			},
			Message: map[string]any{
				"id":     "build-456",
				"status": "SUCCESS",
			},
			Logger: logger,
			Events: directEvents,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, directEvents.Count())
	})

	t.Run("direct build source emits only builds without buildTriggerId", func(t *testing.T) {
		directEvents := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"buildSource": buildSourceDirect,
			},
			Message: map[string]any{
				"id":     "build-456",
				"status": "SUCCESS",
			},
			Logger: logger,
			Events: directEvents,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, directEvents.Count())

		triggeredEvents := &testcontexts.EventContext{}
		err = trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"buildSource": buildSourceDirect,
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-abc",
			},
			Logger: logger,
			Events: triggeredEvents,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, triggeredEvents.Count())
	})

	t.Run("trigger ID filter matches emits event", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"triggerId": "trigger-abc",
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-abc",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("trigger ID filter does not match skips event", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"triggerId": "trigger-abc",
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-xyz",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("combined status and trigger ID filter both must match", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"statuses":  []string{"SUCCESS"},
				"triggerId": "trigger-abc",
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-abc",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

	t.Run("combined filter: status matches but trigger ID does not skips event", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"statuses":  []string{"SUCCESS"},
				"triggerId": "trigger-abc",
			},
			Message: map[string]any{
				"id":             "build-123",
				"status":         "SUCCESS",
				"buildTriggerId": "trigger-xyz",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}
