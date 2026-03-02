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

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"identifier": "Wklm1a2b"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing identifier -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "my-org/my-repo"},
		})

		require.ErrorContains(t, err, "identifier is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:     &contexts.HTTPContext{},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "my-org/my-repo",
				"identifier": "Wklm1a2b",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetPackage__Execute(t *testing.T) {
	component := &GetPackage{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name":"my-lib","version":"1.0.0","format":"python"}`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":    "test-api-key",
				"workspace": "my-org",
			},
		},
		HTTP:           httpCtx,
		ExecutionState: execState,
		Configuration: map[string]any{
			"repository": "my-org/my-repo",
			"identifier": "Wklm1a2b",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
	assert.Equal(t, "cloudsmith.package", execState.Type)
	require.Len(t, execState.Payloads, 1)
}
