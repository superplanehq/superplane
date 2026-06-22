package storage

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteBucket__Setup(t *testing.T) {
	d := &DeleteBucket{}
	setup := func(cfg map[string]any) error {
		return d.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing bucket -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "bucket is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"bucket": "my-bucket"}))
	})
}

func Test__DeleteBucket__Execute(t *testing.T) {
	d := &DeleteBucket{}

	t.Run("deletes the bucket and confirms deletion", func(t *testing.T) {
		var deleteURL string
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte(``), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "my-bucket"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.storage.bucket", state.Type)
		assert.Contains(t, deleteURL, "/b/my-bucket")

		data := firstData(t, state)
		assert.Equal(t, "my-bucket", data["name"])
		assert.Equal(t, true, data["deleted"])
	})

	t.Run("treats a missing bucket as already deleted", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "gone"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		data := firstData(t, state)
		assert.Equal(t, "gone", data["name"])
		assert.Equal(t, true, data["deleted"])
	})

	t.Run("surfaces a non-404 API error", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, fmt.Errorf("boom")
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "my-bucket"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to delete bucket")
	})
}
