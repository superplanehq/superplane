package contexts

import (
	"fmt"

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

	statesMap := make(map[string]models.CapabilityState)
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

func (c *CapabilityContext) Enable(capabilities ...string) error {
	for _, capability := range capabilities {
		_, ok := c.definitions[capability]
		if !ok {
			return fmt.Errorf("capability %s not found", capability)
		}

		c.states[capability] = models.CapabilityState{Name: capability, State: core.IntegrationCapabilityStateEnabled}
	}

	return nil
}

func (c *CapabilityContext) Disable(capabilities ...string) error {
	for _, capability := range capabilities {
		_, ok := c.definitions[capability]
		if !ok {
			return fmt.Errorf("capability %s not found", capability)
		}

		c.states[capability] = models.CapabilityState{Name: capability, State: core.IntegrationCapabilityStateDisabled}
	}

	return nil
}

func (c *CapabilityContext) IsRequested(capabilities ...string) (bool, error) {
	for _, capability := range capabilities {
		v, ok := c.states[capability]
		if !ok {
			return false, fmt.Errorf("capability %s not found", capability)
		}

		if v.State != core.IntegrationCapabilityStateRequested {
			return false, nil
		}
	}

	return true, nil
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
