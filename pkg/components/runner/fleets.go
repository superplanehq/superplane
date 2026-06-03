package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// ComponentName is the registry / canvas component key for Runner.
const ComponentName = "runner"

const configurationFieldMachineType = "machine_type"

// User-facing machine type names (select labels).
const (
	MachineTypeE1LargeAMD64 = "e1-large-amd64"
	MachineTypeE1LargeARM64 = "e1-large-arm64"
)

// Task-broker fleet IDs (stored in node configuration and sent as fleet_id).
const (
	brokerFleetAWSStandard1 = "aws-standard-1"
	brokerFleetAWSARM64_1   = "aws-arm64-1"
)

// Hardcoded machine types mapped to current broker fleets (GET /v1/fleets).
var machineTypeSelectOptions = []configuration.FieldOption{
	{Label: MachineTypeE1LargeAMD64, Value: brokerFleetAWSStandard1},
	{Label: MachineTypeE1LargeARM64, Value: brokerFleetAWSARM64_1},
}

// MachineTypeLabel returns the user-facing name for a broker fleet ID, or the ID if unknown.
func MachineTypeLabel(fleetID string) string {
	switch strings.TrimSpace(fleetID) {
	case brokerFleetAWSStandard1:
		return MachineTypeE1LargeAMD64
	case brokerFleetAWSARM64_1:
		return MachineTypeE1LargeARM64
	default:
		return fleetID
	}
}

func requireMachineType(machineType string) (string, error) {
	fleet := strings.TrimSpace(machineType)
	if fleet == "" {
		return "", fmt.Errorf("machine type is required")
	}
	return fleet, nil
}
