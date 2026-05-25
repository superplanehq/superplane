package core

import "slices"

// BuildCapabilities returns Capability metadata derived from integration actions and triggers.
func BuildCapabilities(actions []Action, triggers []Trigger) []Capability {
	capabilities := make([]Capability, 0, len(actions)+len(triggers))
	for _, action := range actions {
		capabilities = append(capabilities, Capability{
			Type:           IntegrationCapabilityTypeAction,
			Name:           action.Name(),
			Label:          action.Label(),
			Description:    action.Description(),
			Configuration:  action.Configuration(),
			OutputChannels: action.OutputChannels(nil),
		})
	}
	for _, trigger := range triggers {
		capabilities = append(capabilities, Capability{
			Type:          IntegrationCapabilityTypeTrigger,
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Configuration: trigger.Configuration(),
		})
	}
	return capabilities
}

// CapabilityNamesNotRequested returns capability names referenced in capability groups but missing
// from requestedCapabilities (capabilities the user chose at capability selection).
func CapabilityNamesNotRequested(groups []CapabilityGroup, requestedCapabilities []string) []string {
	diff := make([]string, 0)
	for _, group := range groups {
		for _, capability := range group.Capabilities {
			if !slices.Contains(requestedCapabilities, capability.Name) {
				diff = append(diff, capability.Name)
			}
		}
	}
	return diff
}
