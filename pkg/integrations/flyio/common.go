package flyio

import "strings"

const (
	// machinePollInterval is the interval between machine state polls.
	machinePollInterval = 30

	// maxMachinePollAttempts is the maximum number of poll attempts before
	// giving up and routing to the failed output channel. At 30 s/attempt
	// this gives a 15-minute ceiling before a machine is considered stuck.
	maxMachinePollAttempts = 30
)

// parseMachineID extracts the bare machine ID from a composite "appName/machineID"
// string that the IntegrationResource picker stores as the resource ID.
// If the string isn't compound, it is returned as-is.
func parseMachineID(compound string) string {
	parts := strings.SplitN(compound, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return compound
}

// machinePayload converts a Machine into a workflow output payload map.
func machinePayload(appName string, m *Machine) map[string]any {
	payload := map[string]any{
		"machineId": m.ID,
		"appName":   appName,
		"state":     m.State,
		"region":    m.Region,
	}

	if m.Config != nil {
		payload["image"] = m.Config.Image
	}

	return payload
}
