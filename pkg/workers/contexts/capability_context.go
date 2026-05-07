package contexts

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type CapabilityContext struct {
	definitions map[string]core.Capability
	states      map[string]models.CapabilityState
}

func NewCapabilityContext(definitions []core.Capability, states []models.CapabilityState) *CapabilityContext {
	definitionsMap := make(map[string]core.Capability)
	for _, definition := range definitions {
		definitionsMap[definition.Name] = definition
	}

	statesMap := map[string]models.CapabilityState{}
	for _, state := range states {
		statesMap[state.Name] = state
	}

	return &CapabilityContext{
		definitions: definitionsMap,
		states:      statesMap,
	}
}

func (c *CapabilityContext) States() []models.CapabilityState {
	states := make([]models.CapabilityState, 0, len(c.states))
	for _, state := range c.states {
		states = append(states, state)
	}
	return states
}

func (c *CapabilityContext) updateState(newState core.IntegrationCapabilityState, capabilities ...string) {
	for _, capability := range capabilities {
		c.states[capability] = models.CapabilityState{Name: capability, State: newState}
	}
}

func (c *CapabilityContext) Request(capabilities ...string) {
	c.updateState(core.IntegrationCapabilityStateRequested, capabilities...)
}

func (c *CapabilityContext) Enable(capabilities ...string) {
	c.updateState(core.IntegrationCapabilityStateEnabled, capabilities...)
}

func (c *CapabilityContext) Disable(capabilities ...string) {
	c.updateState(core.IntegrationCapabilityStateDisabled, capabilities...)
}

func (c *CapabilityContext) Available(capabilities ...string) {
	c.updateState(core.IntegrationCapabilityStateAvailable, capabilities...)
}

func (c *CapabilityContext) Unavailable(capabilities ...string) {
	c.updateState(core.IntegrationCapabilityStateUnavailable, capabilities...)
}

func (c *CapabilityContext) Clear() {
	c.states = map[string]models.CapabilityState{}
}

func (c *CapabilityContext) IsRequested(capabilities ...string) bool {
	for _, capability := range capabilities {
		v, ok := c.states[capability]
		if !ok {
			return false
		}

		if v.State != core.IntegrationCapabilityStateRequested {
			return false
		}
	}

	return true
}

func (c *CapabilityContext) Requested() []string {
	requested := []string{}
	for _, capability := range c.states {
		if capability.State == core.IntegrationCapabilityStateRequested {
			requested = append(requested, capability.Name)
		}
	}
	return requested
}

func (c *CapabilityContext) Enabled() []string {
	enabled := []string{}
	for _, capability := range c.states {
		if capability.State == core.IntegrationCapabilityStateEnabled {
			enabled = append(enabled, capability.Name)
		}
	}
	return enabled
}
