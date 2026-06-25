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

func Test__ScanPackage__Setup(t *testing.T) {
	component := &ScanPackage{}

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
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
				},
			},
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
		assert.Equal(t, "my-package 1.0.0", metadata.PackageName)
		assert.Equal(t, "perm123", metadata.PackageID)
	})
}

func Test__ScanPackage__Execute(t *testing.T) {
	component := &ScanPackage{}

	t.Run("successful scan schedules scan and emits confirmation with package name", func(t *testing.T) {
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
						Body:       io.NopCloser(strings.NewReader(`{}`)),
					},
					okResponse(`{"slug":"my-package-1-0-0","slug_perm":"perm123","name":"my-package","version":"1.0.0"}`),
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
		assert.Equal(t, "cloudsmith.package.scan_scheduled", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		result, ok := wrapped["data"].(ScanPackageResult)
		require.True(t, ok)
		assert.Equal(t, "acme/production", result.Repository)
		assert.Equal(t, "perm123", result.Package)
		assert.Equal(t, "my-package", result.Name)
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

	t.Run("API error returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(strings.NewReader(`{"detail":"Package not found."}`)),
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to schedule scan")
		assert.False(t, executionState.Passed)
	})
}
