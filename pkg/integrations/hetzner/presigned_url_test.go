package hetzner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestPresignedURL_Setup(t *testing.T) {
	component := &PresignedURL{}

	t.Run("missing method -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key",
			},
		})
		require.ErrorContains(t, err, "method is required")
	})

	t.Run("invalid method -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "method": "DELETE",
			},
		})
		require.ErrorContains(t, err, "method must be GET or PUT")
	})

	t.Run("valid GET method -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "method": "GET",
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid PUT method, lowercase -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Configuration: map[string]any{
				"bucket": "my-bucket", "key": "my-key", "method": "put",
			},
		})
		require.NoError(t, err)
	})
}
