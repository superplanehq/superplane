package cloudstorage

import (
	"context"
	"encoding/json"
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
	assert.NotEmpty(t, payload["bucket"])
	assert.NotEmpty(t, payload["name"])
}

func TestUploadObject_Setup(t *testing.T) {
	c := &UploadObject{}

	t.Run("stores bucket and object in metadata", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"object":  "reports/output.json",
				"content": `{"key":"value"}`,
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		stored := meta.Get().(UploadObjectMetadata)
		assert.Equal(t, "my-bucket", stored.Bucket)
		assert.Equal(t, "reports/output.json", stored.Object)
	})

	t.Run("returns error when bucket is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"object":  "file.txt",
				"content": "data",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("returns error when object is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"content": "data",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "object is required")
	})
}

func TestUploadObject_Execute(t *testing.T) {
	t.Run("uploads content and emits result", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
					assert.Contains(t, fullURL, "/b/my-bucket/o")
					assert.Contains(t, fullURL, "name=reports%2Foutput.json")
					assert.Equal(t, `{"key":"value"}`, body)
					return json.Marshal(map[string]any{
						"bucket":      "my-bucket",
						"name":        "reports/output.json",
						"size":        "15",
						"contentType": "application/json",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&UploadObject{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":      "my-bucket",
				"object":      "reports/output.json",
				"content":     `{"key":"value"}`,
				"contentType": "application/json",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "my-bucket", data["bucket"])
		assert.Equal(t, "reports/output.json", data["name"])
	})

	t.Run("fails when API returns error", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return nil, assert.AnError
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&UploadObject{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":  "my-bucket",
				"object":  "file.txt",
				"content": "hello",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to upload object")
	})
}
