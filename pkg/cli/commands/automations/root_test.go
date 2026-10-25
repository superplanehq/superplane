package automations

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func TestHelpDoesNotRegisterAutomationDefinitionEdits(t *testing.T) {
	root := NewCommand(core.BindOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})

	require.NoError(t, root.Execute())
	help := buf.String()

	for _, name := range []string{"list", "active", "delete"} {
		require.Contains(t, help, name)
	}
	for _, name := range []string{"create", "get", "init", "update", "staging"} {
		require.NotContains(t, help, name)
		_, _, err := root.Find([]string{name})
		require.Error(t, err)
	}
}
