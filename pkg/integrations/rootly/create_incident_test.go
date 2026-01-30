package rootly

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title":    "Test Incident",
				"summary":  "Test summary",
				"severity": "sev1",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing title returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"summary":  "Test summary",
				"severity": "sev1",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("empty title returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title":    "",
				"summary":  "Test summary",
				"severity": "sev1",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("title only - optional fields not required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "Minimal Incident",
			},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}
