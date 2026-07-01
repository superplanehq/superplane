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

		ctx.Enable("deploy")
		ctx.Disable("rollback")

		states := ctx.States()
		assertCapabilityState(t, states, "deploy", core.IntegrationCapabilityStateEnabled)
		assertCapabilityState(t, states, "rollback", core.IntegrationCapabilityStateDisabled)
	})

	t.Run("checks requested capabilities", func(t *testing.T) {
		ctx := NewCapabilityContext(definitions, []models.CapabilityState{
			{Name: "deploy", State: core.IntegrationCapabilityStateRequested},
			{Name: "rollback", State: core.IntegrationCapabilityStateEnabled},
			{Name: "promote", State: core.IntegrationCapabilityStateRequested},
		})

		assert.True(t, ctx.IsRequested("deploy", "promote"))
		assert.False(t, ctx.IsRequested("deploy", "rollback"))
		assert.False(t, ctx.IsRequested("missing"))
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
