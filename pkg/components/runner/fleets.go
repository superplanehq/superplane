package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// ComponentName is the registry / canvas component key for Runner.
const ComponentName = "runner"

const configurationFieldMachineType = "machine_type"

// Machine type names, which are also the default task-broker fleet IDs
// (GET /v1/fleets) stored in node configuration and sent as fleet_id
// unless TASK_BROKER_FLEET_ID is set.
const (
	MachineTypeE1LargeAMD64 = "e1-large-amd64"
	MachineTypeE1LargeARM64 = "e1-large-arm64"
	MachineTypeE1TinyAMD64  = "e1-tiny-amd64"
	MachineTypeE1TinyARM64  = "e1-tiny-arm64"
)

func MachineTypeField(name string) configuration.Field {
	return configuration.Field{
		Name:     name,
		Label:    "Machine type",
		Type:     configuration.FieldTypeRunnerFleet,
		Required: true,
	}
}

func requireMachineType(machineType string) (string, error) {
	fleet := strings.TrimSpace(machineType)
	if fleet == "" {
		return "", fmt.Errorf("machine type is required")
	}
	return fleet, nil
}

// resolveBrokerFleetID is the fleet_id sent to the task broker.
// TASK_BROKER_FLEET_ID overrides the node machine type when set. When it is
// empty, a local Docker broker URL uses fleet local so the sibling worker can
// claim the task.
func resolveBrokerFleetID(machineType string) (string, error) {
	if _, err := requireMachineType(machineType); err != nil {
		return "", err
	}
	if fleet := strings.TrimSpace(os.Getenv("TASK_BROKER_FLEET_ID")); fleet != "" {
		return fleet, nil
	}
	if isLocalTaskBrokerURL(os.Getenv("TASK_BROKER_BASE_URL")) {
		return localComposeFleetID, nil
	}
	return strings.TrimSpace(machineType), nil
}
