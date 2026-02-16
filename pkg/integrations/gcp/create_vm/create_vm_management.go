package createvm

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	compute "google.golang.org/api/compute/v1"
)

const (
	OnHostMaintenanceMigrate   = "MIGRATE"
	OnHostMaintenanceTerminate = "TERMINATE"
)

const (
	metadataKeyStartupScript  = "startup-script"
	metadataKeyShutdownScript = "shutdown-script"
)

type ManagementConfig struct {
	MetadataItems     []MetadataKeyValue `mapstructure:"metadataItems"`
	StartupScript     string             `mapstructure:"startupScript"`
	ShutdownScript    string             `mapstructure:"shutdownScript"`
	AutomaticRestart  *bool              `mapstructure:"automaticRestart"`
	OnHostMaintenance string             `mapstructure:"onHostMaintenance"`
	MaintenancePolicy string             `mapstructure:"maintenancePolicy"`
}

type MetadataKeyValue struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

func CreateVMManagementConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "metadataItems",
			Label:       "Custom metadata",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional key-value metadata for the instance.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Metadata",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Metadata key (e.g. my-config, role).",
								Placeholder: "e.g. my-config",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Metadata value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
		{
			Name:        "startupScript",
			Label:       "Startup script (optional)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Script that runs when the instance boots.",
			Placeholder: "#!/bin/bash\necho 'Hello from startup script'",
		},
		{
			Name:        "shutdownScript",
			Label:       "Shutdown script (optional)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Script that runs when the instance is shut down.",
			Placeholder: "#!/bin/bash\necho 'Goodbye from shutdown script'",
		},
		{
			Name:        "automaticRestart",
			Label:       "Automatic restart",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Restart the VM automatically when it crashes or is terminated by the system.",
			Default:     true,
		},
		{
			Name:        "onHostMaintenance",
			Label:       "On host maintenance",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "When the host undergoes maintenance: migrate the VM to another host, or terminate it.",
			Default:     OnHostMaintenanceMigrate,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Migrate VM (recommended)", Value: OnHostMaintenanceMigrate},
						{Label: "Terminate VM", Value: OnHostMaintenanceTerminate},
					},
				},
			},
		},
		{
			Name:        "maintenancePolicy",
			Label:       "Maintenance policy",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional resource policy URL for instance scheduling (e.g. start/stop windows).",
			Placeholder: "e.g. projects/my-project/regions/region/resourcePolicies/my-policy",
		},
	}
}

func BuildInstanceMetadata(config ManagementConfig) *compute.Metadata {
	var items []*compute.MetadataItems
	seen := make(map[string]bool)

	if script := strings.TrimSpace(config.StartupScript); script != "" {
		items = append(items, &compute.MetadataItems{Key: metadataKeyStartupScript, Value: &script})
		seen[metadataKeyStartupScript] = true
	}
	if script := strings.TrimSpace(config.ShutdownScript); script != "" {
		items = append(items, &compute.MetadataItems{Key: metadataKeyShutdownScript, Value: &script})
		seen[metadataKeyShutdownScript] = true
	}

	for _, m := range config.MetadataItems {
		k := strings.TrimSpace(m.Key)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		v := strings.TrimSpace(m.Value)
		vCopy := v
		items = append(items, &compute.MetadataItems{Key: k, Value: &vCopy})
	}

	if len(items) == 0 {
		return nil
	}
	return &compute.Metadata{Items: items}
}

func BuildScheduling(config ManagementConfig) *compute.Scheduling {
	automaticRestart := true
	if config.AutomaticRestart != nil {
		automaticRestart = *config.AutomaticRestart
	}
	onHostMaintenance := OnHostMaintenanceMigrate
	if strings.TrimSpace(config.OnHostMaintenance) == OnHostMaintenanceTerminate {
		onHostMaintenance = OnHostMaintenanceTerminate
	}
	return &compute.Scheduling{
		AutomaticRestart:  &automaticRestart,
		OnHostMaintenance: onHostMaintenance,
	}
}
