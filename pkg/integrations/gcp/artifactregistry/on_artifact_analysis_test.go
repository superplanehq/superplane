package artifactregistry

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnArtifactAnalysisOnIntegrationMessage(t *testing.T) {
	trigger := &OnArtifactAnalysis{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("default mode emits only discovery occurrences", func(t *testing.T) {
		client := &mockClient{
			projectID: "demo-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return []byte(`{"name":"projects/demo-project/occurrences/occ-1","kind":"DISCOVERY","resourceUri":"https://us-central1-docker.pkg.dev/demo-project/my-repo/my-image@sha256:abc123","discovery":{"analysisStatus":"FINISHED_SUCCESS"}}`), nil
			},
		}
		setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return client, nil
		})

		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"name": "projects/demo-project/occurrences/occ-1",
				"kind": "DISCOVERY",
			},
			Integration: &testcontexts.IntegrationContext{},
			Logger:      logger,
			Events:      events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, ArtifactAnalysisEmittedEventType, events.Payloads[0].Type)
	})

	t.Run("package filter skips malformed resource URI", func(t *testing.T) {
		client := &mockClient{
			projectID: "demo-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return []byte(`{"name":"projects/demo-project/occurrences/occ-2","kind":"VULNERABILITY","resourceUri":"https://us-central1-docker.pkg.dev/demo-project/my-repo"}`), nil
			},
		}
		setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return client, nil
		})

		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"kinds": []string{"VULNERABILITY"}, "package": "my-image"},
			Message: map[string]any{
				"name": "projects/demo-project/occurrences/occ-2",
				"kind": "VULNERABILITY",
			},
			Integration: &testcontexts.IntegrationContext{},
			Logger:      logger,
			Events:      events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}
