package cloudstorage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetObject_Metadata(t *testing.T) {
	c := &GetObject{}
	assert.Equal(t, "gcp.cloudstorage.getObject", c.Name())
	assert.Equal(t, "Cloud Storage • Get Object", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.NotEmpty(t, c.Documentation())
	assert.Equal(t, "gcp", c.Icon())
	assert.Equal(t, "gray", c.Color())
	assert.Nil(t, c.Actions())
}

func TestGetObject_ExampleOutput(t *testing.T) {
	c := &GetObject{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["timestamp"])
	payload, ok := output["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, payload["name"])
	assert.NotEmpty(t, payload["bucket"])
}

func TestGetObject_Setup(t *testing.T) {
	c := &GetObject{}

	t.Run("stores bucket and object in metadata", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket": "my-bucket",
				"object": "path/to/file.json",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		stored := meta.Get().(GetObjectMetadata)
		assert.Equal(t, "my-bucket", stored.Bucket)
		assert.Equal(t, "path/to/file.json", stored.Object)
	})

	t.Run("returns error when bucket is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"object": "file.json",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("returns error when object is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket": "my-bucket",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "object is required")
	})
}

func TestGetObject_Execute(t *testing.T) {
	t.Run("gets object metadata and emits result", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, fullURL string) ([]byte, error) {
					assert.Contains(t, fullURL, "/b/my-bucket/o/data%2Freport.csv")
					return json.Marshal(map[string]any{
						"name":         "data/report.csv",
						"bucket":       "my-bucket",
						"size":         "1048576",
						"contentType":  "text/csv",
						"timeCreated":  "2025-01-01T12:00:00.000Z",
						"updated":      "2025-01-01T12:00:00.000Z",
						"storageClass": "STANDARD",
						"md5Hash":      "abc123==",
						"generation":   "1735689600000000",
						"selfLink":     "https://www.googleapis.com/storage/v1/b/my-bucket/o/data%2Freport.csv",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&GetObject{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "my-bucket", "object": "data/report.csv"},
			ExecutionState: state,
			NodeMetadata: &testcontexts.MetadataContext{Metadata: GetObjectMetadata{
				Bucket: "my-bucket",
				Object: "data/report.csv",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "data/report.csv", data["name"])
		assert.Equal(t, "my-bucket", data["bucket"])
		assert.Equal(t, "1048576", data["size"])
		assert.Equal(t, "text/csv", data["contentType"])
	})

	t.Run("fails when API returns error", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return nil, errors.New("object not found")
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&GetObject{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"bucket": "my-bucket", "object": "missing.json"},
			ExecutionState: state,
			NodeMetadata: &testcontexts.MetadataContext{Metadata: GetObjectMetadata{
				Bucket: "my-bucket",
				Object: "missing.json",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "object not found")
	})
}
