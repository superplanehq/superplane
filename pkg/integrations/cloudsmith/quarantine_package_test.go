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

func Test__QuarantinePackage__Setup(t *testing.T) {
	component := &QuarantinePackage{}

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
				"action":     QuarantineActionQuarantine,
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

	t.Run("invalid action returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     "Delete",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "action must be")
	})

	t.Run("valid quarantine configuration resolves package metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     QuarantineActionQuarantine,
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
	})

	t.Run("valid release configuration resolves package metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     QuarantineActionRelease,
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
	})
}

func Test__QuarantinePackage__Execute(t *testing.T) {
	component := &QuarantinePackage{}

	quarantinedPackageJSON := `{
		"slug": "my-package-1-0-0",
		"slug_perm": "perm123",
		"name": "my-package",
		"version": "1.0.0",
		"format": "python",
		"status": 8,
		"status_str": "Quarantined",
		"repository": "production",
		"namespace": "acme",
		"cdn_url": "https://dl.cloudsmith.io/public/acme/production/python/my-package-1.0.0.tar.gz",
		"self_html_url": "https://cloudsmith.io/~acme/repos/production/packages/detail/python/my-package/1.0.0/"
	}`

	releasedPackageJSON := `{
		"slug": "my-package-1-0-0",
		"slug_perm": "perm123",
		"name": "my-package",
		"version": "1.0.0",
		"format": "python",
		"status": 2,
		"status_str": "Available",
		"repository": "production",
		"namespace": "acme",
		"cdn_url": "https://dl.cloudsmith.io/public/acme/production/python/my-package-1.0.0.tar.gz"
	}`

	t.Run("quarantine action emits quarantined package", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     QuarantineActionQuarantine,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(quarantinedPackageJSON)),
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
		assert.Equal(t, "cloudsmith.package.quarantined", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		pkg, ok := wrapped["data"].(*Package)
		require.True(t, ok)
		assert.Equal(t, "my-package", pkg.Name)
		assert.Equal(t, "Quarantined", pkg.StatusStr)
	})

	t.Run("release action emits released package", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     QuarantineActionRelease,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(releasedPackageJSON)),
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
		assert.Equal(t, "cloudsmith.package.released", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		pkg, ok := wrapped["data"].(*Package)
		require.True(t, ok)
		assert.Equal(t, "Available", pkg.StatusStr)
	})

	t.Run("invalid repository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "no-namespace",
				"package":    "perm123",
				"action":     QuarantineActionQuarantine,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository")
	})

	t.Run("API error returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
				"action":     QuarantineActionQuarantine,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Body:       io.NopCloser(strings.NewReader(`{"detail":"Permission denied."}`)),
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to quarantine package")
	})
}
