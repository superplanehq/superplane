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

func Test__DeleteTag__Setup(t *testing.T) {
	component := &DeleteTag{}

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
			Configuration: map[string]any{"tag": "v1.0.0"},
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

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:     &contexts.HTTPContext{},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "superplane/demo",
				"tag":        "v1.0.0",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteTag__Execute(t *testing.T) {
	component := &DeleteTag{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				accessTokenSecretName: {Name: accessTokenSecretName, Value: []byte("token")},
			},
		},
		HTTP:           httpCtx,
		ExecutionState: execState,
		Configuration: map[string]any{
			"repository": "superplane/demo",
			"tag":        "v1.0.0",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
	assert.Equal(t, "dockerhub.deletedTag", execState.Type)
	require.Len(t, execState.Payloads, 1)
}
