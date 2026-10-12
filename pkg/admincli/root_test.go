package admincli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandIncludesAdminNamespaces(t *testing.T) {
	for _, path := range [][]string{
		{"fleets", "list"},
		{"fleets", "describe"},
		{"fleets", "create"},
		{"fleets", "update"},
		{"tasks", "list"},
		{"runners", "list"},
		{"runners", "describe"},
		{"runners", "delete"},
	} {
		command, remaining, err := RootCmd.Find(path)
		require.NoError(t, err)
		assert.Empty(t, remaining)
		assert.Equal(t, path[len(path)-1], command.Name())
	}
}

func TestRootCommandIncludesConnectionFlags(t *testing.T) {
	for _, name := range []string{"config", "url", "token", "output", "verbose"} {
		assert.NotNil(t, RootCmd.PersistentFlags().Lookup(name), name)
	}
}
