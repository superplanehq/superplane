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

func Test__PromotePackage__Setup(t *testing.T) {
	component := &PromotePackage{}

	t.Run("missing sourceRepository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"package":               "perm123",
				"destinationRepository": "acme/production",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "sourceRepository is required")
	})

	t.Run("missing package returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"destinationRepository": "acme/production",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Staging","slug":"staging","namespace":"acme"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.ErrorContains(t, err, "package is required")
	})

	t.Run("missing destinationRepository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceRepository": "acme/staging",
				"package":          "perm123",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Staging","slug":"staging","namespace":"acme"}`),
					okResponse(`{"slug":"my-package-1-0-0","slug_perm":"perm123","name":"my-package","version":"1.0.0"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.ErrorContains(t, err, "destinationRepository is required")
	})

	t.Run("valid configuration resolves package metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "acme/production",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Staging","slug":"staging","namespace":"acme"}`),
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
		assert.Equal(t, "Staging", metadata.RepositoryName)
		assert.Equal(t, "perm123", metadata.PackageID)
	})
}

func Test__PromotePackage__Execute(t *testing.T) {
	component := &PromotePackage{}

	promotedPackageJSON := `{
		"slug": "my-package-1-0-0",
		"slug_perm": "perm123",
		"name": "my-package",
		"version": "1.0.0",
		"format": "docker",
		"status": 2,
		"status_str": "Available",
		"repository": "production",
		"namespace": "acme",
		"size": 52428800,
		"size_str": "50.0 MB",
		"self_webapp_url": "https://app.cloudsmith.com/acme/r/production/docker/my-package/1.0.0/perm123"
	}`

	t.Run("copy mode emits promoted package data", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "acme/production",
				"mode":                  PromoteModeCopy,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(promotedPackageJSON)),
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
		assert.Equal(t, "cloudsmith.package.promoted", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		pkg, ok := wrapped["data"].(*Package)
		require.True(t, ok)
		assert.Equal(t, "my-package", pkg.Name)
		assert.Equal(t, "1.0.0", pkg.Version)
		assert.Equal(t, "production", pkg.Repository)
	})

	t.Run("move mode emits promoted package data", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "acme/production",
				"mode":                  PromoteModeMove,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(promotedPackageJSON)),
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
		assert.Equal(t, "cloudsmith.package.promoted", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})

	t.Run("invalid sourceRepository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "invalid",
				"package":               "perm123",
				"destinationRepository": "acme/production",
				"mode":                  PromoteModeCopy,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid sourceRepository")
	})

	t.Run("invalid destinationRepository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "invalid",
				"mode":                  PromoteModeCopy,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid destinationRepository")
	})

	t.Run("cross-namespace destination returns error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "other-owner/production",
				"mode":                  PromoteModeCopy,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cross-namespace promotion is not supported")
	})

	t.Run("API error returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sourceRepository":      "acme/staging",
				"package":               "perm123",
				"destinationRepository": "acme/production",
				"mode":                  PromoteModeCopy,
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
		assert.Contains(t, err.Error(), "failed to copy package")
	})
}

func Test__PromotePackage__ExampleOutput(t *testing.T) {
	output := (&PromotePackage{}).ExampleOutput()
	require.NotNil(t, output)
	assert.Equal(t, "cloudsmith.package.promoted", output["type"])
	assert.NotEmpty(t, output["timestamp"])
	assert.NotNil(t, output["data"])
}
