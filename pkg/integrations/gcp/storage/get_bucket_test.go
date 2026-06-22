package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetBucket__Setup(t *testing.T) {
	g := &GetBucket{}
	setup := func(cfg map[string]any) error {
		return g.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing bucket -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "bucket is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"bucket": "my-bucket"}))
	})
}

func Test__GetBucket__Execute(t *testing.T) {
	g := &GetBucket{}

	t.Run("fetches the bucket and emits its details", func(t *testing.T) {
		var getURL string
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				getURL = url
				return []byte(`{"id":"my-bucket","name":"my-bucket","location":"EU","locationType":"multi-region","storageClass":"NEARLINE","timeCreated":"2025-01-01T00:00:00.000Z","selfLink":"https://www.googleapis.com/storage/v1/b/my-bucket","versioning":{"enabled":true}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "my-bucket"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.storage.bucket", state.Type)
		assert.Contains(t, getURL, "/b/my-bucket")

		data := firstData(t, state)
		assert.Equal(t, "EU", data["location"])
		assert.Equal(t, "NEARLINE", data["storageClass"])
		assert.Equal(t, true, data["versioning"])
		assert.Equal(t, "https://console.cloud.google.com/storage/browser/my-bucket", data["consoleUrl"])
	})

	t.Run("surfaces an API error", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "missing"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get bucket")
	})
}
