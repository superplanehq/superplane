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

var machineTypeSelectOptions = []configuration.FieldOption{
	{Label: MachineTypeE1LargeAMD64, Value: MachineTypeE1LargeAMD64},
	{Label: MachineTypeE1LargeARM64, Value: MachineTypeE1LargeARM64},
	{Label: MachineTypeE1TinyAMD64, Value: MachineTypeE1TinyAMD64},
	{Label: MachineTypeE1TinyARM64, Value: MachineTypeE1TinyARM64},
}

func requireMachineType(machineType string) (string, error) {
	fleet := strings.TrimSpace(machineType)
	if fleet == "" {
		return "", fmt.Errorf("machine type is required")
	}
	return fleet, nil
}

// resolveBrokerFleetID is the fleet_id sent to the task broker.
// TASK_BROKER_FLEET_ID overrides the node machine type when set, so a local
// broker with one fleet (for example local) can run nodes that still
// select e1-large-amd64 in the canvas.
func resolveBrokerFleetID(machineType string) (string, error) {
	if _, err := requireMachineType(machineType); err != nil {
		return "", err
	}
	if fleet := strings.TrimSpace(os.Getenv("TASK_BROKER_FLEET_ID")); fleet != "" {
		return fleet, nil
	}
	return strings.TrimSpace(machineType), nil
}
