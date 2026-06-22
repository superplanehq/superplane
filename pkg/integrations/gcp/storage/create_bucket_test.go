package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateBucket__Setup(t *testing.T) {
	c := &CreateBucket{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing name -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"location": "US"}), "name is required")
	})

	t.Run("missing location -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "my-bucket"}), "location is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"name": "my-bucket", "location": "US"}))
	})
}

func Test__CreateBucket__Execute(t *testing.T) {
	c := &CreateBucket{}

	t.Run("creates the bucket and emits its details", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				raw, _ := json.Marshal(body)
				_ = json.Unmarshal(raw, &postBody)
				return []byte(`{"id":"my-bucket","name":"my-bucket","location":"US","locationType":"multi-region","storageClass":"STANDARD","timeCreated":"2025-01-01T00:00:00.000Z","selfLink":"https://www.googleapis.com/storage/v1/b/my-bucket","iamConfiguration":{"uniformBucketLevelAccess":{"enabled":true}}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":         "my-bucket",
				"location":     "US",
				"storageClass": "STANDARD",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.storage.bucket", state.Type)
		assert.Contains(t, postURL, "/b?project=my-project")
		assert.Equal(t, "my-bucket", postBody["name"])
		assert.Equal(t, "US", postBody["location"])

		data := firstData(t, state)
		assert.Equal(t, "my-bucket", data["name"])
		assert.Equal(t, "US", data["location"])
		assert.Equal(t, "STANDARD", data["storageClass"])
		assert.Equal(t, true, data["uniformBucketLevelAccess"])
		assert.Equal(t, "https://console.cloud.google.com/storage/browser/my-bucket", data["consoleUrl"])
	})

	t.Run("enables uniform bucket-level access by default", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				raw, _ := json.Marshal(body)
				_ = json.Unmarshal(raw, &postBody)
				return []byte(`{"name":"my-bucket","location":"US"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "my-bucket", "location": "US"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		iam, ok := postBody["iamConfiguration"].(map[string]any)
		require.True(t, ok, "expected iamConfiguration in request body")
		ubla := iam["uniformBucketLevelAccess"].(map[string]any)
		assert.Equal(t, true, ubla["enabled"])
	})

	t.Run("surfaces an API error", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				return nil, fmt.Errorf("boom")
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "my-bucket", "location": "US"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to create bucket")
	})
}
