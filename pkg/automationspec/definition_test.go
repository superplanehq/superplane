package automationspec_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automationspec"
)

func TestWorkspaceSourceSwitchPreservesModelOnlyYAML(t *testing.T) {
	source := definitionHeader + "agent: {model: old-model}\n" + manualTrigger + bashStep
	switched := &automationspec.Agent{
		Harness: automationspec.Codex, Credentials: &automationspec.Credentials{Source: automationspec.Subscription},
	}
	updated, err := automationspec.ReplaceExplicitAgent(source, switched)
	require.NoError(t, err)
	document, err := automationspec.Parse(updated)
	require.NoError(t, err)
	require.Nil(t, document.Agent)

	switched.Model = "new-model"
	updated, err = automationspec.ReplaceExplicitAgent(source, switched)
	require.NoError(t, err)
	document, err = automationspec.Parse(updated)
	require.NoError(t, err)
	require.Equal(t, &automationspec.Agent{Model: "new-model"}, document.Agent)
}
