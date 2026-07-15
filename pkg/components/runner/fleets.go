package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// ComponentName is the registry / canvas component key for Runner.
const ComponentName = "runner"

const configurationFieldMachineType = "machine_type"

// Machine type names, which are also the task-broker fleet IDs (GET /v1/fleets)
// stored in node configuration and sent as fleet_id.
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

// MachineTypeSelectOptions returns a copy of the machine-type options so other
// components that offload compute to a runner can render the same fleet picker.
func MachineTypeSelectOptions() []configuration.FieldOption {
	options := make([]configuration.FieldOption, len(machineTypeSelectOptions))
	copy(options, machineTypeSelectOptions)
	return options
}

func requireMachineType(machineType string) (string, error) {
	fleet := strings.TrimSpace(machineType)
	if fleet == "" {
		return "", fmt.Errorf("machine type is required")
	}
	return fleet, nil
}
