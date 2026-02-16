package createvm

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	compute "google.golang.org/api/compute/v1"
)

const (
	NodeAffinityOperatorIn    = "IN"
	NodeAffinityOperatorNotIn = "NOT_IN"
)

type AdvancedConfig struct {
	GuestAccelerators      []GuestAcceleratorEntry `mapstructure:"guestAccelerators"`
	NodeAffinities         []NodeAffinityEntry     `mapstructure:"nodeAffinities"`
	ResourcePolicies       []string                `mapstructure:"resourcePolicies"`
	MinNodeCpus            int64                   `mapstructure:"minNodeCpus"`
	Labels                 []LabelEntry            `mapstructure:"labels"`
	EnableDisplayDevice    bool                    `mapstructure:"enableDisplayDevice"`
	EnableSerialPortAccess bool                    `mapstructure:"enableSerialPortAccess"`
}

type LabelEntry struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

type GuestAcceleratorEntry struct {
	AcceleratorType  string `mapstructure:"acceleratorType"`
	AcceleratorCount int64  `mapstructure:"acceleratorCount"`
}

type NodeAffinityEntry struct {
	Key      string   `mapstructure:"key"`
	Operator string   `mapstructure:"operator"`
	Values   []string `mapstructure:"values"`
}

func trimmedNonEmptyStrings(ss []string) []string {
	var out []string
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func normalizeNodeAffinityOperator(op string) string {
	if strings.TrimSpace(op) == NodeAffinityOperatorNotIn {
		return NodeAffinityOperatorNotIn
	}
	return NodeAffinityOperatorIn
}

func BuildGuestAccelerators(config AdvancedConfig) []*compute.AcceleratorConfig {
	var out []*compute.AcceleratorConfig
	for _, e := range config.GuestAccelerators {
		t := strings.TrimSpace(e.AcceleratorType)
		if t == "" || e.AcceleratorCount < 1 {
			continue
		}
		out = append(out, &compute.AcceleratorConfig{
			AcceleratorType:  t,
			AcceleratorCount: e.AcceleratorCount,
		})
	}
	return out
}

func BuildNodeAffinities(config AdvancedConfig) []*compute.SchedulingNodeAffinity {
	var out []*compute.SchedulingNodeAffinity
	for _, e := range config.NodeAffinities {
		key := strings.TrimSpace(e.Key)
		values := trimmedNonEmptyStrings(e.Values)
		if key == "" || len(values) == 0 {
			continue
		}
		op := normalizeNodeAffinityOperator(e.Operator)
		out = append(out, &compute.SchedulingNodeAffinity{
			Key:      key,
			Operator: op,
			Values:   values,
		})
	}
	return out
}

func BuildInstanceResourcePolicies(config AdvancedConfig) []string {
	return trimmedNonEmptyStrings(config.ResourcePolicies)
}

func BuildLabels(config AdvancedConfig) map[string]string {
	if len(config.Labels) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, e := range config.Labels {
		k := strings.TrimSpace(e.Key)
		if k == "" || out[k] != "" {
			continue
		}
		out[k] = strings.TrimSpace(e.Value)
	}
	return out
}

func ApplyAdvancedScheduling(s *compute.Scheduling, config AdvancedConfig) {
	if s == nil {
		return
	}
	if affinities := BuildNodeAffinities(config); len(affinities) > 0 {
		s.NodeAffinities = affinities
	}
	if config.MinNodeCpus > 0 {
		s.MinNodeCpus = config.MinNodeCpus
	}
}

func CreateVMAdvancedConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the instance (billing, environment, team).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center).",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
		{
			Name:        "guestAccelerators",
			Label:       "GPU accelerators",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional GPU or other accelerator cards (e.g. NVIDIA T4, V100, A100, L4).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Accelerator",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "acceleratorType",
								Label:       "Accelerator type",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Type name or full URL (e.g. nvidia-tesla-t4, nvidia-l4).",
								Placeholder: "e.g. nvidia-tesla-t4",
							},
							{
								Name:        "acceleratorCount",
								Label:       "Count",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Description: "Number of accelerator cards to attach.",
								Default:     1,
							},
						},
					},
				},
			},
		},
		{
			Name:        "minNodeCpus",
			Label:       "Min node CPUs (placement)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "For sole-tenant: minimum number of virtual CPUs this instance will consume on a node. Leave empty for shared tenancy.",
			Placeholder: "e.g. 4",
		},
		{
			Name:        "nodeAffinities",
			Label:       "Node affinity (sole-tenant / host)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Constrain placement to specific nodes (e.g. sole-tenant node groups).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Affinity rule",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key of the node (e.g. compute.googleapis.com/node-group).",
								Placeholder: "e.g. compute.googleapis.com/node-group",
							},
							{
								Name:        "operator",
								Label:       "Operator",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Description: "IN: instance must run on nodes with one of the values; NOT_IN: avoid those nodes.",
								Default:     NodeAffinityOperatorIn,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "IN (affinity)", Value: NodeAffinityOperatorIn},
											{Label: "NOT IN (anti-affinity)", Value: NodeAffinityOperatorNotIn},
										},
									},
								},
							},
							{
								Name:        "values",
								Label:       "Values",
								Type:        configuration.FieldTypeList,
								Required:    true,
								Description: "Node label values (e.g. node group names) to match.",
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel: "Value",
										ItemDefinition: &configuration.ListItemDefinition{
											Type: configuration.FieldTypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "resourcePolicies",
			Label:       "Resource policies",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Instance resource policy URLs (e.g. for start/stop schedules).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Policy URL",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "enableDisplayDevice",
			Label:       "Enable display device",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Enable a virtual display device for the instance.",
			Default:     false,
		},
		{
			Name:        "enableSerialPortAccess",
			Label:       "Enable serial port access",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Allow connecting to the instance serial console.",
			Default:     false,
		},
	}
}
