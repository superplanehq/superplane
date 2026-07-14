package hetzner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestUploadObject_Setup(t *testing.T) {
	component := &UploadObject{}

	t.Run("missing content -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key",
			},
		})
		require.ErrorContains(t, err, "content is required")
	})

	t.Run("empty string content -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "content": "",
			},
		})
		require.ErrorContains(t, err, "content is required")
	})

	t.Run("non-empty string content -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "content": "hello",
			},
		})
		require.NoError(t, err)
	})

	t.Run("falsy but explicit content (0) -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "content": float64(0),
			},
		})
		require.NoError(t, err)
	})
}
