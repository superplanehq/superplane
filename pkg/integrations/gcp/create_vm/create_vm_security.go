package createvm

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	compute "google.golang.org/api/compute/v1"
)

const (
	ConfidentialInstanceTypeSEV    = "SEV"     // AMD Secure Encrypted Virtualization
	ConfidentialInstanceTypeSEVSNP = "SEV_SNP" // AMD SEV - Secure Nested Paging
	ConfidentialInstanceTypeTDX    = "TDX"     // Intel Trust Domain eXtension
)

const (
	fieldNameShieldedVM     = "shieldedVM"
	fieldNameConfidentialVM = "confidentialVM"
)

var (
	visibleWhenShieldedVM = []configuration.VisibilityCondition{
		{Field: fieldNameShieldedVM, Values: []string{"true"}},
	}
	visibleWhenConfidentialVM = []configuration.VisibilityCondition{
		{Field: fieldNameConfidentialVM, Values: []string{"true"}},
	}
)

type SecurityConfig struct {
	ShieldedVM                          bool   `mapstructure:"shieldedVM"`
	ShieldedVMEnableSecureBoot          bool   `mapstructure:"shieldedVMEnableSecureBoot"`
	ShieldedVMEnableVtpm                bool   `mapstructure:"shieldedVMEnableVtpm"`
	ShieldedVMEnableIntegrityMonitoring bool   `mapstructure:"shieldedVMEnableIntegrityMonitoring"`
	ConfidentialVM                      bool   `mapstructure:"confidentialVM"`
	ConfidentialVMType                  string `mapstructure:"confidentialVMType"`
}

func CreateVMSecurityConfigFields() []configuration.Field {
	shielded := shieldedVMFields()
	confidential := confidentialVMFields()
	return append(shielded, confidential...)
}

func shieldedVMFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        fieldNameShieldedVM,
			Label:       "Shielded VM",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Use Shielded VM for verified boot and measured boot. Enables vTPM and integrity monitoring by default; you can optionally enable Secure Boot.",
			Default:     false,
		},
		{
			Name:                 "shieldedVMEnableSecureBoot",
			Label:                "Secure Boot",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Verify digital signatures of all boot components. Disabled by default due to possible compatibility issues with unsigned drivers.",
			Default:              false,
			VisibilityConditions: visibleWhenShieldedVM,
		},
		{
			Name:                 "shieldedVMEnableVtpm",
			Label:                "vTPM",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Virtual Trusted Platform Module for measured boot and key storage.",
			Default:              true,
			VisibilityConditions: visibleWhenShieldedVM,
		},
		{
			Name:                 "shieldedVMEnableIntegrityMonitoring",
			Label:                "Integrity monitoring",
			Type:                 configuration.FieldTypeBool,
			Required:             false,
			Description:          "Monitor boot integrity against a baseline from the trusted boot image.",
			Default:              true,
			VisibilityConditions: visibleWhenShieldedVM,
		},
	}
}

func confidentialVMFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        fieldNameConfidentialVM,
			Label:       "Confidential VM",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Run the VM with Confidential Computing (memory encrypted by the host). Requires a supported machine type (e.g. N2D, C2D).",
			Default:     false,
		},
		{
			Name:                 "confidentialVMType",
			Label:                "Confidential instance type",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Description:          "Technology used for confidential compute. SEV (AMD) is common; SEV-SNP and TDX (Intel) depend on machine type and availability.",
			Default:              ConfidentialInstanceTypeSEV,
			VisibilityConditions: visibleWhenConfidentialVM,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: confidentialInstanceTypeOptions(),
				},
			},
		},
	}
}

func confidentialInstanceTypeOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "AMD SEV", Value: ConfidentialInstanceTypeSEV},
		{Label: "AMD SEV-SNP", Value: ConfidentialInstanceTypeSEVSNP},
		{Label: "Intel TDX", Value: ConfidentialInstanceTypeTDX},
	}
}

func BuildShieldedInstanceConfig(config SecurityConfig) *compute.ShieldedInstanceConfig {
	if !config.ShieldedVM {
		return nil
	}
	return &compute.ShieldedInstanceConfig{
		EnableSecureBoot:          config.ShieldedVMEnableSecureBoot,
		EnableVtpm:                config.ShieldedVMEnableVtpm,
		EnableIntegrityMonitoring: config.ShieldedVMEnableIntegrityMonitoring,
	}
}

func BuildConfidentialInstanceConfig(config SecurityConfig) *compute.ConfidentialInstanceConfig {
	if !config.ConfidentialVM {
		return nil
	}
	confidentialType := config.ConfidentialVMType
	if confidentialType == "" {
		confidentialType = ConfidentialInstanceTypeSEV
	}
	return &compute.ConfidentialInstanceConfig{
		EnableConfidentialCompute: true,
		ConfidentialInstanceType:  confidentialType,
	}
}
