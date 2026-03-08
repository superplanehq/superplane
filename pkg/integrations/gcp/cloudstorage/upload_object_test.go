package cloudstorage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestUploadObject_Metadata(t *testing.T) {
	c := &UploadObject{}
	assert.Equal(t, "gcp.cloudstorage.uploadObject", c.Name())
	assert.Equal(t, "Cloud Storage • Upload Object", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.NotEmpty(t, c.Documentation())
	assert.Equal(t, "gcp", c.Icon())
	assert.Equal(t, "gray", c.Color())
	assert.Nil(t, c.Actions())
}

func TestUploadObject_ExampleOutput(t *testing.T) {
	c := &UploadObject{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["timestamp"])
	payload, ok := output["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, payload["name"])
	assert.NotEmpty(t, payload["bucket"])
}

func TestUploadObject_Setup(t *testing.T) {
	c := &UploadObject{}

	t.Run("stores bucket and object in metadata", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"object":  "output/results.json",
				"content": `{"key":"value"}`,
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		stored := meta.Get().(UploadObjectMetadata)
		assert.Equal(t, "my-bucket", stored.Bucket)
		assert.Equal(t, "output/results.json", stored.Object)
	})

	t.Run("returns error when bucket is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"object":  "file.json",
				"content": "hello",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("returns error when object path is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"content": "hello",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "object path is required")
	})
}

func TestUploadObject_Execute(t *testing.T) {
	t.Run("uploads content and emits result", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				execRequest: func(_ context.Context, method, url string, body io.Reader) ([]byte, error) {
					assert.Equal(t, "POST", method)
					assert.Contains(t, url, "/b/my-bucket/o")
					assert.Contains(t, url, "name=output%2Fresults.json")

					content, _ := io.ReadAll(body)
					assert.Equal(t, `{"key":"value"}`, string(content))

					return json.Marshal(map[string]any{
						"name":         "output/results.json",
						"bucket":       "my-bucket",
						"size":         "15",
						"contentType":  "application/json",
						"timeCreated":  "2025-01-01T12:00:00.000Z",
						"updated":      "2025-01-01T12:00:00.000Z",
						"storageClass": "STANDARD",
						"md5Hash":      "xyz789==",
						"generation":   "1735689600000000",
						"selfLink":     "https://www.googleapis.com/storage/v1/b/my-bucket/o/output%2Fresults.json",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&UploadObject{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":      "my-bucket",
				"object":      "output/results.json",
				"content":     `{"key":"value"}`,
				"contentType": "application/json",
			},
			ExecutionState: state,
			NodeMetadata: &testcontexts.MetadataContext{Metadata: UploadObjectMetadata{
				Bucket: "my-bucket",
				Object: "output/results.json",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "output/results.json", data["name"])
		assert.Equal(t, "my-bucket", data["bucket"])
		assert.Equal(t, "application/json", data["contentType"])
	})

	t.Run("fails when API returns error", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				execRequest: func(_ context.Context, _ string, _ string, _ io.Reader) ([]byte, error) {
					return nil, errors.New("permission denied")
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&UploadObject{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"object":  "output/results.json",
				"content": "hello",
			},
			ExecutionState: state,
			NodeMetadata: &testcontexts.MetadataContext{Metadata: UploadObjectMetadata{
				Bucket: "my-bucket",
				Object: "output/results.json",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "permission denied")
	})
}
