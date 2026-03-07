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

func TestOnArtifactAnalysisSetup(t *testing.T) {
	trigger := &OnArtifactAnalysis{}

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

		metadata := OnArtifactAnalysisMetadata{}
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
			Metadata: OnArtifactAnalysisMetadata{
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

func TestOnArtifactAnalysisMetadata(t *testing.T) {
	trigger := &OnArtifactAnalysis{}
	assert.Equal(t, "gcp.artifactregistry.onArtifactAnalysis", trigger.Name())
	assert.Equal(t, "Artifact Registry • On Artifact Analysis", trigger.Label())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
}

func TestOnArtifactAnalysisExampleData(t *testing.T) {
	trigger := &OnArtifactAnalysis{}
	data := trigger.ExampleData()
	assert.Equal(t, containerAnalysisServiceName, data["serviceName"])
	assert.Equal(t, createOccurrenceMethod, data["methodName"])
	assert.NotEmpty(t, data["resourceName"])
}

func TestOnArtifactAnalysisOnIntegrationMessage(t *testing.T) {
	trigger := &OnArtifactAnalysis{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("emits event for CreateOccurrence", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  containerAnalysisServiceName,
				"methodName":   createOccurrenceMethod,
				"resourceName": "projects/my-project/occurrences/occ-1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("emits event for BatchCreateOccurrences", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  containerAnalysisServiceName,
				"methodName":   batchCreateOccurrenceMethod,
				"resourceName": "projects/my-project/occurrences/occ-2",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("skips non-container-analysis events", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  "compute.googleapis.com",
				"methodName":   createOccurrenceMethod,
				"resourceName": "projects/my-project/instances/vm-1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("skips unrelated container analysis methods", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"serviceName":  containerAnalysisServiceName,
				"methodName":   "google.devtools.containeranalysis.v1.Grafeas.GetOccurrence",
				"resourceName": "projects/my-project/occurrences/occ-1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}
