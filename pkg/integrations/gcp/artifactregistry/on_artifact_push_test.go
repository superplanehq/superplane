package artifactregistry

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnArtifactPushOnIntegrationMessage(t *testing.T) {
	trigger := &OnArtifactPush{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("repository filter matches repository segment", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"repository": "my-repo"},
			Message: map[string]any{
				"action": "INSERT",
				"digest": "us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, ArtifactPushEmittedEventType, events.Payloads[0].Type)
		payload, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123", payload["digest"])
	})

	t.Run("repository filter skips mismatched repository", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"repository": "other-repo"},
			Message: map[string]any{
				"action": "INSERT",
				"digest": "us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("location filter uses exact location", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"location": "europe-west1"},
			Message: map[string]any{
				"action": "INSERT",
				"digest": "us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("non insert action is ignored", func(t *testing.T) {
		events := &testcontexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{},
			Message: map[string]any{
				"action": "DELETE",
				"digest": "us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}
