package rootly

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__UpdateIncident__Setup(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("valid configuration with all fields", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident_id": "abc123",
				"title":       "Updated Title",
				"summary":     "Updated summary",
				"severity":    "sev1",
				"status":      "mitigated",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with incident_id only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident_id": "abc123",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incident_id returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title":  "Updated Title",
				"status": "resolved",
			},
		})

		require.ErrorContains(t, err, "incident_id is required")
	})

	t.Run("empty incident_id returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident_id": "",
				"title":       "Updated Title",
			},
		})

		require.ErrorContains(t, err, "incident_id is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}
