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
	MachineTypeE1TinyAMD64  = "e1-tiny-amd64"
	MachineTypeE1TinyARM64  = "e1-tiny-arm64"
)

// Task-broker fleet IDs (stored in node configuration and sent as fleet_id).
const (
	brokerFleetAWSStandard1 = "aws-standard-1"
	brokerFleetAWSARM64_1   = "aws-arm64-1"
	brokerFleetE1TinyAMD64  = "e1-tiny-amd64"
	brokerFleetE1TinyARM64  = "e1-tiny-arm64"
)

// Hardcoded machine types mapped to current broker fleets (GET /v1/fleets).
var machineTypeSelectOptions = []configuration.FieldOption{
	{Label: MachineTypeE1LargeAMD64, Value: brokerFleetAWSStandard1},
	{Label: MachineTypeE1LargeARM64, Value: brokerFleetAWSARM64_1},
	{Label: MachineTypeE1TinyAMD64, Value: brokerFleetE1TinyAMD64},
	{Label: MachineTypeE1TinyARM64, Value: brokerFleetE1TinyARM64},
}

// MachineTypeLabel returns the user-facing name for a broker fleet ID, or the ID if unknown.
func MachineTypeLabel(fleetID string) string {
	switch strings.TrimSpace(fleetID) {
	case brokerFleetAWSStandard1:
		return MachineTypeE1LargeAMD64
	case brokerFleetAWSARM64_1:
		return MachineTypeE1LargeARM64
	case brokerFleetE1TinyAMD64:
		return MachineTypeE1TinyAMD64
	case brokerFleetE1TinyARM64:
		return MachineTypeE1TinyARM64
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
