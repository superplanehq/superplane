package artifactregistry

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnArtifactPushSetup(t *testing.T) {
	trigger := &OnArtifactPush{}

	t.Run("creates subscription on first setup", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{}
		metadataCtx := &testcontexts.MetadataContext{}
		requestCtx := &testcontexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Requests:    requestCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		metadata := OnArtifactPushMetadata{}
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
		assert.NotEmpty(t, metadata.SubscriptionID)
		assert.NotEmpty(t, metadata.SinkID)
	})

	t.Run("returns error when integration is nil", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "connect the GCP integration")
	})

	t.Run("skips when already configured", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{}
		metadataCtx := &testcontexts.MetadataContext{
			Metadata: OnArtifactPushMetadata{
				SubscriptionID: "existing-sub",
				SinkID:         "existing-sink",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, integrationCtx.Subscriptions, "should not create new subscription when already configured")
	})
}

func TestOnArtifactPushMetadata(t *testing.T) {
	trigger := &OnArtifactPush{}
	assert.Equal(t, "gcp.artifactregistry.onArtifactPush", trigger.Name())
	assert.Equal(t, "Artifact Registry • On Artifact Push", trigger.Label())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
}

func TestOnArtifactPushExampleData(t *testing.T) {
	trigger := &OnArtifactPush{}
	data := trigger.ExampleData()
	assert.Equal(t, artifactRegistryServiceName, data["serviceName"])
	assert.Equal(t, createDockerImageMethod, data["methodName"])
	assert.NotEmpty(t, data["resourceName"])
}

func TestOnArtifactPushOnIntegrationMessage(t *testing.T) {
	trigger := &OnArtifactPush{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("emits event for matching artifact push", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  artifactRegistryServiceName,
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/img",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("skips non-artifact-registry service events", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  "compute.googleapis.com",
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/instances/vm-1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("skips non-create-docker-image method events", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  artifactRegistryServiceName,
				"methodName":   "google.devtools.artifactregistry.v1.ArtifactRegistry.GetDockerImage",
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/img",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("filters by location when configured", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"location": "us-east1",
			},
			Message: map[string]any{
				"serviceName":  artifactRegistryServiceName,
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/img",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count(), "should not emit when location does not match")
	})

	t.Run("filters by repository when configured", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"repository": "other-repo",
			},
			Message: map[string]any{
				"serviceName":  artifactRegistryServiceName,
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/img",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count(), "should not emit when repository does not match")
	})

	t.Run("emits when location and repository match", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"location":   "us-central1",
				"repository": "my-repo",
			},
			Message: map[string]any{
				"serviceName":  artifactRegistryServiceName,
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/img",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}
