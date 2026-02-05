package rootly

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__GetIncident__Setup(t *testing.T) {
	component := &GetIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("empty incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}

func Test__GetIncident__Name(t *testing.T) {
	component := &GetIncident{}
	require.Equal(t, "rootly.getIncident", component.Name())
}

func Test__GetIncident__Label(t *testing.T) {
	component := &GetIncident{}
	require.Equal(t, "Get Incident", component.Label())
}

func Test__GetIncident__Description(t *testing.T) {
	component := &GetIncident{}
	require.Equal(t, "Retrieve incident details from Rootly", component.Description())
}

func Test__GetIncident__Documentation(t *testing.T) {
	component := &GetIncident{}
	doc := component.Documentation()
	require.Contains(t, doc, "Get Incident component")
	require.Contains(t, doc, "Use Cases")
	require.Contains(t, doc, "Configuration")
	require.Contains(t, doc, "Output")
}

func Test__GetIncident__Icon(t *testing.T) {
	component := &GetIncident{}
	require.Equal(t, "alert-triangle", component.Icon())
}

func Test__GetIncident__Color(t *testing.T) {
	component := &GetIncident{}
	require.Equal(t, "gray", component.Color())
}

func Test__GetIncident__OutputChannels(t *testing.T) {
	component := &GetIncident{}
	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	require.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__GetIncident__Configuration(t *testing.T) {
	component := &GetIncident{}
	config := component.Configuration()

	require.Len(t, config, 1)

	incidentIdField := config[0]
	require.Equal(t, "incidentId", incidentIdField.Name)
	require.Equal(t, "Incident ID", incidentIdField.Label)
	require.True(t, incidentIdField.Required)
}

func Test__GetIncident__Cancel(t *testing.T) {
	component := &GetIncident{}
	err := component.Cancel(core.ExecutionContext{})
	require.NoError(t, err)
}

func Test__GetIncident__Cleanup(t *testing.T) {
	component := &GetIncident{}
	err := component.Cleanup(core.SetupContext{})
	require.NoError(t, err)
}

func Test__GetIncident__HandleAction(t *testing.T) {
	component := &GetIncident{}
	err := component.HandleAction(core.ActionContext{})
	require.NoError(t, err)
}

func Test__GetIncident__Actions(t *testing.T) {
	component := &GetIncident{}
	actions := component.Actions()
	require.Empty(t, actions)
}

func Test__GetIncident__HandleWebhook(t *testing.T) {
	component := &GetIncident{}
	status, err := component.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	require.Equal(t, 200, status)
}
