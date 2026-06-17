package cloudsmith

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPackageStatus__Setup(t *testing.T) {
	component := &GetPackageStatus{}

	t.Run("missing repository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing package returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP:     &contexts.HTTPContext{Responses: []*http.Response{okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`)}},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.ErrorContains(t, err, "package is required")
	})

	t.Run("expression repository and package passes without API call", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "{{ $.trigger.data.repository }}",
				"package":    "{{ $.trigger.data.package }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration resolves package metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
					okResponse(`{"slug":"my-package-1-0-0","slug_perm":"perm123","name":"my-package","version":"1.0.0"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(PackageNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Production", metadata.RepositoryName)
		assert.Equal(t, "my-package-1-0-0", metadata.PackageName)
		assert.Equal(t, "perm123", metadata.PackageID)
	})
}

func Test__GetPackageStatus__Execute(t *testing.T) {
	component := &GetPackageStatus{}

	statusJSON := `{
		"self_url": "https://api.cloudsmith.io/v1/packages/acme/production/perm123/status/",
		"stage": 2,
		"stage_str": "Available",
		"stage_updated_at": "2026-01-15T10:00:05.000Z",
		"status": 2,
		"status_reason": "",
		"status_str": "Available",
		"status_updated_at": "2026-01-15T10:00:05.000Z",
		"is_sync_awaiting": false,
		"is_sync_completed": true,
		"is_sync_failed": false,
		"is_sync_in_flight": false,
		"is_sync_in_progress": false,
		"is_quarantined": false,
		"sync_finished_at": "2026-01-15T10:00:05.000Z",
		"sync_progress": 100
	}`

	t.Run("successful fetch emits status data", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(statusJSON)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "cloudsmith.package.status", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		info, ok := wrapped["data"].(*PackageStatusInfo)
		require.True(t, ok)
		assert.Equal(t, "Available", info.StageStr)
		assert.Equal(t, "Available", info.StatusStr)
		assert.True(t, info.IsSyncCompleted)
		assert.False(t, info.IsQuarantined)
		assert.Equal(t, 100, info.SyncProgress)
	})

	t.Run("invalid repository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "no-namespace",
				"package":    "perm123",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository")
		assert.False(t, executionState.Passed)
	})

	t.Run("package not found returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "missing-perm",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader(`{"detail":"Not found."}`)),
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get package status")
		assert.False(t, executionState.Passed)
	})
}
