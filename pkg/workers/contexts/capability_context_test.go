package contexts

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__CapabilityContext(t *testing.T) {
	definitions := []core.Capability{
		{Name: "deploy"},
		{Name: "rollback"},
		{Name: "promote"},
	}

	t.Run("enables and disables defined capabilities", func(t *testing.T) {
		ctx := NewCapabilityContext(definitions, []models.CapabilityState{
			{Name: "deploy", State: core.IntegrationCapabilityStateRequested},
			{Name: "rollback", State: core.IntegrationCapabilityStateEnabled},
		})

		require.NoError(t, ctx.Enable("deploy"))
		require.NoError(t, ctx.Disable("rollback"))

		states := ctx.States()
		assertCapabilityState(t, states, "deploy", core.IntegrationCapabilityStateEnabled)
		assertCapabilityState(t, states, "rollback", core.IntegrationCapabilityStateDisabled)
	})

	t.Run("rejects unknown capabilities on enable and disable", func(t *testing.T) {
		ctx := NewCapabilityContext(definitions, nil)

		err := ctx.Enable("missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "capability missing not found")

		err = ctx.Disable("missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "capability missing not found")
		assert.Empty(t, ctx.States())
	})

	t.Run("checks requested capabilities", func(t *testing.T) {
		ctx := NewCapabilityContext(definitions, []models.CapabilityState{
			{Name: "deploy", State: core.IntegrationCapabilityStateRequested},
			{Name: "rollback", State: core.IntegrationCapabilityStateEnabled},
			{Name: "promote", State: core.IntegrationCapabilityStateRequested},
		})

		requested, err := ctx.IsRequested("deploy", "promote")
		require.NoError(t, err)
		assert.True(t, requested)

		requested, err = ctx.IsRequested("deploy", "rollback")
		require.NoError(t, err)
		assert.False(t, requested)

		requested, err = ctx.IsRequested("missing")
		require.Error(t, err)
		assert.False(t, requested)
		assert.Contains(t, err.Error(), "capability missing not found")
	})

	t.Run("returns requested capability names", func(t *testing.T) {
		ctx := NewCapabilityContext(definitions, []models.CapabilityState{
			{Name: "deploy", State: core.IntegrationCapabilityStateRequested},
			{Name: "rollback", State: core.IntegrationCapabilityStateEnabled},
			{Name: "promote", State: core.IntegrationCapabilityStateRequested},
		})

		requested := ctx.Requested()
		slices.Sort(requested)
		assert.Equal(t, []string{"deploy", "promote"}, requested)
	})
}

func assertCapabilityState(t *testing.T, states []models.CapabilityState, name string, state core.IntegrationCapabilityState) {
	t.Helper()

	index := slices.IndexFunc(states, func(s models.CapabilityState) bool {
		return s.Name == name
	})

	require.NotEqual(t, -1, index)
	assert.Equal(t, state, states[index].State)
}
