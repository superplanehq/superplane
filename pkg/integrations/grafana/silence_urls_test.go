package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__buildSilenceWebURL__usesAlertingSilencePath(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL":  "https://grafana.example.com",
			"apiToken": "t",
		},
	}

	u, err := buildSilenceWebURL(integration, "a3e5c2d1-8b4f-4e1a-9c7d-2f0e6b3a1d5c")
	require.NoError(t, err)
	require.Equal(t, "https://grafana.example.com/alerting/silence/a3e5c2d1-8b4f-4e1a-9c7d-2f0e6b3a1d5c/edit?alertmanager=grafana", u)
}

func Test__buildSilenceWebURL__preservesBasePath(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL":  "https://grafana.example.com/grafana",
			"apiToken": "t",
		},
	}

	u, err := buildSilenceWebURL(integration, "s1")
	require.NoError(t, err)
	require.Equal(t, "https://grafana.example.com/grafana/alerting/silence/s1/edit?alertmanager=grafana", u)
}
