package dockerhub

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

func Test__GetImageTag__Setup(t *testing.T) {
	component := &GetImageTag{}

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
			Configuration: map[string]any{"tag": "latest"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing tag -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "superplane/demo"},
		})

		require.ErrorContains(t, err, "tag is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:     &contexts.HTTPContext{},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "superplane/demo",
				"tag":        "latest",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetImageTag__Execute(t *testing.T) {
	component := &GetImageTag{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":1,"name":"latest"}`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Integration: &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				accessTokenSecretName: {Name: accessTokenSecretName, Value: []byte("token")},
			},
		},
		HTTP:           httpCtx,
		ExecutionState: execState,
		Configuration: map[string]any{
			"repository": "superplane/demo",
			"tag":        "latest",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
	assert.Equal(t, "dockerhub.tag", execState.Type)
	require.Len(t, execState.Payloads, 1)
}
