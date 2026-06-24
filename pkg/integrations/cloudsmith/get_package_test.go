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

func Test__GetPackage__Setup(t *testing.T) {
	component := &GetPackage{}

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

	t.Run("static repository with expression package resolves repository name", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "{{ $.trigger.data.package }}",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
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
		assert.Equal(t, "acme", metadata.RepositoryNamespace)
		assert.Equal(t, "{{ $.trigger.data.package }}", metadata.PackageName)
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

func Test__GetPackage__Execute(t *testing.T) {
	component := &GetPackage{}

	packageJSON := `{
		"slug": "my-package-1-0-0",
		"slug_perm": "perm123",
		"name": "my-package",
		"version": "1.0.0",
		"format": "python",
		"status": 2,
		"status_str": "Available",
		"repository": "production",
		"namespace": "acme",
		"uploaded_at": "2026-01-15T10:00:00.000Z",
		"checksum_md5": "d41d8cd98f00b204e9800998ecf8427e",
		"checksum_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"size": 524288,
		"size_str": "512.0 KB",
		"cdn_url": "https://dl.cloudsmith.io/public/acme/production/python/my-package-1.0.0.tar.gz",
		"self_html_url": "https://cloudsmith.io/~acme/repos/production/packages/detail/python/my-package/1.0.0/"
	}`

	t.Run("successful fetch emits full package data", func(t *testing.T) {
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
						Body:       io.NopCloser(strings.NewReader(packageJSON)),
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
		assert.Equal(t, "cloudsmith.package.fetched", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		pkg, ok := wrapped["data"].(*Package)
		require.True(t, ok)
		assert.Equal(t, "my-package", pkg.Name)
		assert.Equal(t, "1.0.0", pkg.Version)
		assert.Equal(t, "python", pkg.Format)
		assert.Equal(t, "Available", pkg.StatusStr)
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", pkg.ChecksumSHA256)
		assert.Equal(t, "512.0 KB", pkg.SizeStr)
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
		assert.Contains(t, err.Error(), "failed to get package")
		assert.False(t, executionState.Passed)
	})
}
