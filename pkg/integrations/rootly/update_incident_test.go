package rootly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__UpdateIncident__Name(t *testing.T) {
	component := &UpdateIncident{}
	assert.Equal(t, "rootly.updateIncident", component.Name())
}

func Test__UpdateIncident__Label(t *testing.T) {
	component := &UpdateIncident{}
	assert.Equal(t, "Update Incident", component.Label())
}

func Test__UpdateIncident__OutputChannels(t *testing.T) {
	component := &UpdateIncident{}
	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__UpdateIncident__Configuration(t *testing.T) {
	component := &UpdateIncident{}
	fields := component.Configuration()

	require.Len(t, fields, 5)

	// Incident ID field
	assert.Equal(t, "incidentId", fields[0].Name)
	assert.True(t, fields[0].Required)

	// Title field
	assert.Equal(t, "title", fields[1].Name)
	assert.False(t, fields[1].Required)

	// Summary field
	assert.Equal(t, "summary", fields[2].Name)
	assert.False(t, fields[2].Required)

	// Status field
	assert.Equal(t, "status", fields[3].Name)
	assert.False(t, fields[3].Required)

	// Severity field
	assert.Equal(t, "severity", fields[4].Name)
	assert.False(t, fields[4].Required)
}

func Test__UpdateIncident__Setup(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("incident ID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("accepts valid incident ID", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "2b4a5c6d-7e8f-9a0b-c1d2-e3f4a5b6c7d8",
			},
		})

		require.NoError(t, err)
	})

	t.Run("accepts incident ID with optional fields", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "2b4a5c6d-7e8f-9a0b-c1d2-e3f4a5b6c7d8",
				"title":      "Updated title",
				"summary":    "Updated summary",
				"status":     "mitigated",
				"severity":   "critical",
			},
		})

		require.NoError(t, err)
	})
}
