package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateImage__Setup(t *testing.T) {
	component := &CreateImage{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"sourceType": "disk", "sourceDisk": "my-disk"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "image name is required")
	})

	t.Run("disk source without disk returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "img", "sourceType": "disk"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "source disk is required")
	})

	t.Run("snapshot source without snapshot returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "img", "sourceType": "snapshot"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "source snapshot is required")
	})

	t.Run("disk source by name without zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "img", "sourceType": "disk", "sourceDisk": "my-disk"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("disk source by full path without zone is allowed", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":       "img",
				"sourceType": "disk",
				"sourceDisk": "projects/my-project/zones/us-central1-a/disks/my-disk",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid disk source stores image name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "img", "sourceType": "disk", "zone": "us-central1-a", "sourceDisk": "my-disk"},
			Metadata:      meta,
		})
		require.NoError(t, err)
		var stored ImageNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "img", stored.ImageName)
	})

	t.Run("empty source type defaults to disk", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "img", "zone": "us-central1-a", "sourceDisk": "my-disk"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__CreateImage__Execute(t *testing.T) {
	component := &CreateImage{}

	t.Run("creates image from disk -> emits created event", func(t *testing.T) {
		var postPath string
		var postBody any
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postPath = path
				postBody = body
				return opDone("op-create"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-create"), nil
				}
				return imageGetJSON("img", "READY", "my-app", map[string]string{"env": "prod"}, "fp", ""), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":       "img",
				"sourceType": "disk",
				"zone":       "us-central1-a",
				"sourceDisk": "my-disk",
				"family":     "my-app",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.image.created", state.Type)
		assert.True(t, strings.HasSuffix(postPath, "/global/images"))
		require.Len(t, state.Payloads, 1)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "img", data["name"])
		assert.Equal(t, "READY", data["status"])

		// the source disk should be resolved to a zone-qualified URL
		bodyMap := map[string]any{}
		require.NoError(t, mapstructure.Decode(postBody, &bodyMap))
		assert.Contains(t, bodyMap["SourceDisk"], "zones/us-central1-a/disks/my-disk")
	})

	t.Run("forceCreate appends query param", func(t *testing.T) {
		var postPath string
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postPath = path
				return opDone("op-create"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-create"), nil
				}
				return imageGetJSON("img", "READY", "", nil, "fp", ""), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":        "img",
				"sourceType":  "disk",
				"zone":        "us-central1-a",
				"sourceDisk":  "my-disk",
				"forceCreate": true,
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.Contains(t, postPath, "forceCreate=true")
	})

	t.Run("multiple storage locations -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "img",
				"sourceType":       "disk",
				"zone":             "us-central1-a",
				"sourceDisk":       "my-disk",
				"storageLocations": "us,eu",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "only one storage location")
	})

	t.Run("missing source -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "img", "sourceType": "disk"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "source disk is required")
	})

	t.Run("API error on create -> fails execution", func(t *testing.T) {
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				return []byte("not-json"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "img", "sourceType": "snapshot", "sourceSnapshot": "snap-1",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "create image operation response")
	})
}
