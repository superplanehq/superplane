package runner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	configurationFieldEnableServerless = "enableServerless"
	configurationFieldFunctionType     = "functionType"
)

// Function type names, which are also the serverless function sizes sent as function_type.
const (
	FunctionType512MB = "512mb"
	FunctionType1GB   = "1gb"
	FunctionType2GB   = "2gb"
	FunctionType4GB   = "4gb"
)

var functionTypeSelectOptions = []configuration.FieldOption{
	{Label: "512 MB", Value: FunctionType512MB},
	{Label: "1 GB", Value: FunctionType1GB},
	{Label: "2 GB", Value: FunctionType2GB},
	{Label: "4 GB", Value: FunctionType4GB},
}

var machineExecutionOnly = []configuration.VisibilityCondition{
	{Field: configurationFieldEnableServerless, Values: []string{"false"}},
}

var serverlessOnly = []configuration.VisibilityCondition{
	{Field: configurationFieldEnableServerless, Values: []string{"true"}},
}

func serverlessConfigurationFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        configurationFieldEnableServerless,
			Label:       "Serverless",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Run as a serverless function instead of on a runner machine.",
		},
		{
			Name:                 configurationFieldFunctionType,
			Label:                "Function type",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              FunctionType512MB,
			Description:          "Memory size of the serverless function.",
			VisibilityConditions: serverlessOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: configurationFieldEnableServerless, Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: functionTypeSelectOptions,
				},
			},
		},
	}
}

func machineTypeConfigurationField() configuration.Field {
	return configuration.Field{
		Name:                 configurationFieldMachineType,
		Label:                "Machine type",
		Type:                 configuration.FieldTypeSelect,
		Required:             false,
		Description:          "Runner fleet the step runs on.",
		VisibilityConditions: machineExecutionOnly,
		RequiredConditions: []configuration.RequiredCondition{
			{Field: configurationFieldEnableServerless, Values: []string{"false"}},
		},
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: machineTypeSelectOptions,
			},
		},
	}
}

// validateComputeSelection enforces exactly one compute target: a serverless function type, or a machine type.
func validateComputeSelection(enableServerless bool, functionType string, machineType string) error {
	if !enableServerless {
		_, err := requireMachineType(machineType)
		return err
	}

	selected := strings.TrimSpace(functionType)
	if selected == "" {
		return fmt.Errorf("function type is required when serverless is enabled")
	}

	known := slices.ContainsFunc(functionTypeSelectOptions, func(option configuration.FieldOption) bool {
		return option.Value == selected
	})
	if !known {
		return fmt.Errorf("invalid function type %q", selected)
	}

	return nil
}

func resolvedFunctionType(enableServerless bool, functionType string) string {
	if !enableServerless {
		return ""
	}
	return strings.TrimSpace(functionType)
}
