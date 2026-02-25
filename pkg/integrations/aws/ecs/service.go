package ecs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	serviceSchedulingStrategyReplica = "REPLICA"
	serviceSchedulingStrategyDaemon  = "DAEMON"
)

var serviceSchedulingStrategyOptions = []configuration.FieldOption{
	{Label: "REPLICA", Value: serviceSchedulingStrategyReplica},
	{Label: "DAEMON", Value: serviceSchedulingStrategyDaemon},
}

var servicePropagateTagOptions = []configuration.FieldOption{
	{Label: "NONE", Value: "NONE"},
	{Label: "SERVICE", Value: "SERVICE"},
	{Label: "TASK_DEFINITION", Value: "TASK_DEFINITION"},
}

var serviceAvailabilityZoneRebalancingOptions = []configuration.FieldOption{
	{Label: "ENABLED", Value: "ENABLED"},
	{Label: "DISABLED", Value: "DISABLED"},
}

var serviceCapacityProviderOptions = []configuration.FieldOption{
	{Label: "EC2", Value: "EC2"},
	{Label: "FARGATE", Value: "FARGATE"},
	{Label: "EXTERNAL", Value: "EXTERNAL"},
	{Label: "MANAGED_INSTANCES", Value: "MANAGED_INSTANCES"},
}

type ServiceMutationConfiguration struct {
	Region                            string                                `json:"region" mapstructure:"region"`
	Cluster                           string                                `json:"cluster" mapstructure:"cluster"`
	TaskDefinition                    string                                `json:"taskDefinition" mapstructure:"taskDefinition"`
	DesiredCount                      *int                                  `json:"desiredCount" mapstructure:"desiredCount"`
	LaunchType                        string                                `json:"launchType" mapstructure:"launchType"`
	CapacityProviderStrategy          []RunTaskCapacityProviderStrategyItem `json:"capacityProviderStrategy" mapstructure:"capacityProviderStrategy"`
	PlatformVersion                   string                                `json:"platformVersion" mapstructure:"platformVersion"`
	EnableExecuteCommand              *bool                                 `json:"enableExecuteCommand" mapstructure:"enableExecuteCommand"`
	NetworkConfiguration              RunTaskNetworkConfiguration           `json:"networkConfiguration" mapstructure:"networkConfiguration"`
	DeploymentConfiguration           map[string]any                        `json:"deploymentConfiguration" mapstructure:"deploymentConfiguration"`
	HealthCheckGracePeriodSeconds     *int                                  `json:"healthCheckGracePeriodSeconds" mapstructure:"healthCheckGracePeriodSeconds"`
	ServiceRegistries                 []map[string]any                      `json:"serviceRegistries" mapstructure:"serviceRegistries"`
	LoadBalancers                     []map[string]any                      `json:"loadBalancers" mapstructure:"loadBalancers"`
	PlacementConstraints              []map[string]any                      `json:"placementConstraints" mapstructure:"placementConstraints"`
	PlacementStrategy                 []map[string]any                      `json:"placementStrategy" mapstructure:"placementStrategy"`
	ServiceConnectConfiguration       map[string]any                        `json:"serviceConnectConfiguration" mapstructure:"serviceConnectConfiguration"`
	VolumeConfigurations              []map[string]any                      `json:"volumeConfigurations" mapstructure:"volumeConfigurations"`
	VpcLatticeConfigurations          []map[string]any                      `json:"vpcLatticeConfigurations" mapstructure:"vpcLatticeConfigurations"`
	AvailabilityZoneRebalancing       string                                `json:"availabilityZoneRebalancing" mapstructure:"availabilityZoneRebalancing"`
	EnableECSManagedTags              *bool                                 `json:"enableECSManagedTags" mapstructure:"enableECSManagedTags"`
	PropagateTags                     string                                `json:"propagateTags" mapstructure:"propagateTags"`
	Tags                              []common.Tag                          `json:"tags" mapstructure:"tags"`
	ForceNewDeployment                *bool                                 `json:"forceNewDeployment" mapstructure:"forceNewDeployment"`
	AdditionalCreateOrUpdateArguments map[string]any                        `json:"additionalCreateOrUpdateArguments" mapstructure:"additionalCreateOrUpdateArguments"`
}

func (config ServiceMutationConfiguration) normalize() ServiceMutationConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.TaskDefinition = strings.TrimSpace(config.TaskDefinition)
	config.LaunchType = strings.ToUpper(strings.TrimSpace(config.LaunchType))
	if config.LaunchType == "AUTO" {
		config.LaunchType = ""
	}
	config.PlatformVersion = strings.TrimSpace(config.PlatformVersion)
	config.PropagateTags = strings.ToUpper(strings.TrimSpace(config.PropagateTags))
	config.AvailabilityZoneRebalancing = strings.ToUpper(strings.TrimSpace(config.AvailabilityZoneRebalancing))

	for i := range config.CapacityProviderStrategy {
		config.CapacityProviderStrategy[i].CapacityProvider = strings.TrimSpace(config.CapacityProviderStrategy[i].CapacityProvider)
	}

	for i := range config.Tags {
		config.Tags[i].Key = strings.TrimSpace(config.Tags[i].Key)
		config.Tags[i].Value = strings.TrimSpace(config.Tags[i].Value)
	}

	if config.NetworkConfiguration.AwsvpcConfiguration != nil {
		config.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIP = strings.TrimSpace(config.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIP)
	}

	return config
}

func (config ServiceMutationConfiguration) validateBase() error {
	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.Cluster == "" {
		return fmt.Errorf("cluster is required")
	}
	if config.DesiredCount != nil && *config.DesiredCount < 0 {
		return fmt.Errorf("desired count cannot be negative")
	}
	if config.HealthCheckGracePeriodSeconds != nil && *config.HealthCheckGracePeriodSeconds < 0 {
		return fmt.Errorf("health check grace period seconds cannot be negative")
	}
	if config.LaunchType != "" && len(config.CapacityProviderStrategy) > 0 {
		return fmt.Errorf("launch type cannot be combined with capacity provider strategy")
	}

	for _, strategy := range config.CapacityProviderStrategy {
		if strategy.CapacityProvider == "" {
			return fmt.Errorf("capacity provider is required for each strategy item")
		}
		if strategy.Weight < 0 {
			return fmt.Errorf("capacity provider weight cannot be negative")
		}
		if strategy.Base < 0 {
			return fmt.Errorf("capacity provider base cannot be negative")
		}
	}

	if config.PropagateTags != "" && !slices.Contains([]string{"NONE", "SERVICE", "TASK_DEFINITION"}, config.PropagateTags) {
		return fmt.Errorf("invalid propagate tags value: %s", config.PropagateTags)
	}

	if config.AvailabilityZoneRebalancing != "" &&
		!slices.Contains([]string{"ENABLED", "DISABLED"}, config.AvailabilityZoneRebalancing) {
		return fmt.Errorf("invalid availability zone rebalancing value: %s", config.AvailabilityZoneRebalancing)
	}

	for _, tag := range config.Tags {
		if tag.Key == "" {
			return fmt.Errorf("tag key is required")
		}
	}

	return nil
}

func (config ServiceMutationConfiguration) toInput() ServiceMutationInput {
	return ServiceMutationInput{
		Cluster:                           config.Cluster,
		TaskDefinition:                    config.TaskDefinition,
		DesiredCount:                      config.DesiredCount,
		LaunchType:                        config.LaunchType,
		CapacityProviderStrategy:          config.CapacityProviderStrategy,
		PlatformVersion:                   config.PlatformVersion,
		EnableExecuteCommand:              config.EnableExecuteCommand,
		NetworkConfiguration:              config.NetworkConfiguration,
		DeploymentConfiguration:           config.DeploymentConfiguration,
		ServiceRegistries:                 config.ServiceRegistries,
		LoadBalancers:                     config.LoadBalancers,
		PlacementConstraints:              config.PlacementConstraints,
		PlacementStrategy:                 config.PlacementStrategy,
		ServiceConnectConfiguration:       config.ServiceConnectConfiguration,
		VolumeConfigurations:              config.VolumeConfigurations,
		VpcLatticeConfigurations:          config.VpcLatticeConfigurations,
		AvailabilityZoneRebalancing:       config.AvailabilityZoneRebalancing,
		HealthCheckGracePeriodSeconds:     config.HealthCheckGracePeriodSeconds,
		EnableECSManagedTags:              config.EnableECSManagedTags,
		PropagateTags:                     config.PropagateTags,
		Tags:                              config.Tags,
		ForceNewDeployment:                config.ForceNewDeployment,
		AdditionalCreateOrUpdateArguments: config.AdditionalCreateOrUpdateArguments,
	}
}

func ecsRegionField() configuration.Field {
	return configuration.Field{
		Name:        "region",
		Label:       "Region",
		Type:        configuration.FieldTypeSelect,
		Required:    true,
		Default:     "us-east-1",
		Description: "AWS region where the ECS cluster is located",
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: common.AllRegions,
			},
		},
	}
}

func ecsClusterField() configuration.Field {
	return configuration.Field{
		Name:        "cluster",
		Label:       "Cluster",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "ECS cluster to run the service or task in",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           "ecs.cluster",
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "region",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "region",
						},
					},
				},
			},
		},
	}
}

func ecsTaskDefinitionField(required bool) configuration.Field {
	return configuration.Field{
		Name:        "taskDefinition",
		Label:       "Task Definition",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    required,
		Togglable:   !required,
		Description: "Task definition family and revision (e.g. myapp:1) to run",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           "ecs.taskDefinition",
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "region",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "region",
						},
					},
				},
			},
		},
	}
}

func ecsServiceMutationFields(defaultDesiredCount any, desiredCountTogglable bool, includeForceNewDeployment bool) []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "desiredCount",
			Label:       "Desired Count",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultDesiredCount,
			Togglable:   desiredCountTogglable,
			Description: "Number of tasks ECS should keep running",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "cluster", Values: []string{"*"}},
			},
		},
		{
			Name:     "launchType",
			Label:    "Launch Type",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "AUTO",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: launchTypeOptions,
				},
			},
		},
		{
			Name:        "capacityProviderStrategy",
			Label:       "Capacity Provider Strategy",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Optional capacity provider strategy (cannot be used with Launch Type)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Strategy",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "capacityProvider",
								Label:    "Capacity Provider",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: serviceCapacityProviderOptions,
									},
								},
							},
							{
								Name:     "weight",
								Label:    "Weight",
								Type:     configuration.FieldTypeNumber,
								Required: false,
								Default:  0,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 0; return &min }(),
									},
								},
							},
							{
								Name:     "base",
								Label:    "Base",
								Type:     configuration.FieldTypeNumber,
								Required: false,
								Default:  0,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 0; return &min }(),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "platformVersion",
			Label:       "Platform Version",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Platform version for Fargate tasks",
		},
		{
			Name:        "enableExecuteCommand",
			Label:       "Enable Execute Command",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Allow ECS Exec on tasks launched by this service",
		},
		{
			Name:        "networkConfiguration",
			Label:       "Network Configuration",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{\"awsvpcConfiguration\":{\"subnets\":[],\"securityGroups\":[],\"assignPublicIp\":\"DISABLED\"}}",
			Togglable:   true,
			Description: "ECS network configuration object (for example awsvpcConfiguration)",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "awsvpcConfiguration",
							Label:    "AWS VPC Configuration",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "subnets",
											Label:    "Subnets",
											Type:     configuration.FieldTypeList,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												List: &configuration.ListTypeOptions{
													ItemLabel: "Subnet",
													ItemDefinition: &configuration.ListItemDefinition{
														Type: configuration.FieldTypeString,
													},
												},
											},
										},
										{
											Name:     "securityGroups",
											Label:    "Security Groups",
											Type:     configuration.FieldTypeList,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												List: &configuration.ListTypeOptions{
													ItemLabel: "Security Group",
													ItemDefinition: &configuration.ListItemDefinition{
														Type: configuration.FieldTypeString,
													},
												},
											},
										},
										{
											Name:     "assignPublicIp",
											Label:    "Assign Public IP",
											Type:     configuration.FieldTypeSelect,
											Required: false,
											Default:  "DISABLED",
											TypeOptions: &configuration.TypeOptions{
												Select: &configuration.SelectTypeOptions{
													Options: []configuration.FieldOption{
														{Label: "Disabled", Value: "DISABLED"},
														{Label: "Enabled", Value: "ENABLED"},
													},
												},
											},
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
			Name:        "deploymentConfiguration",
			Label:       "Deployment Configuration",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{}",
			Togglable:   true,
			Description: "Advanced ECS deployment configuration object",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "alarms",
							Label:    "Alarms",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "alarmNames",
											Label:    "Alarm Names",
											Type:     configuration.FieldTypeList,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												List: &configuration.ListTypeOptions{
													ItemLabel: "Alarm Name",
													ItemDefinition: &configuration.ListItemDefinition{
														Type: configuration.FieldTypeString,
													},
												},
											},
										},
										{
											Name:     "enable",
											Label:    "Enable",
											Type:     configuration.FieldTypeBool,
											Required: false,
										},
										{
											Name:     "rollback",
											Label:    "Rollback",
											Type:     configuration.FieldTypeBool,
											Required: false,
										},
									},
								},
							},
						},
						{
							Name:     "deploymentCircuitBreaker",
							Label:    "Deployment Circuit Breaker",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "enable",
											Label:    "Enable",
											Type:     configuration.FieldTypeBool,
											Required: false,
										},
										{
											Name:     "rollback",
											Label:    "Rollback",
											Type:     configuration.FieldTypeBool,
											Required: false,
										},
									},
								},
							},
						},
						{
							Name:     "maximumPercent",
							Label:    "Maximum Percent",
							Type:     configuration.FieldTypeNumber,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Number: &configuration.NumberTypeOptions{
									Min: func() *int { min := 0; return &min }(),
									Max: func() *int { max := 200; return &max }(),
								},
							},
						},
						{
							Name:     "minimumHealthyPercent",
							Label:    "Minimum Healthy Percent",
							Type:     configuration.FieldTypeNumber,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Number: &configuration.NumberTypeOptions{
									Min: func() *int { min := 0; return &min }(),
									Max: func() *int { max := 100; return &max }(),
								},
							},
						},
						{
							Name:     "strategy",
							Label:    "Strategy",
							Type:     configuration.FieldTypeSelect,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "ROLLING", Value: "ROLLING"},
										{Label: "BLUE_GREEN", Value: "BLUE_GREEN"},
										{Label: "LINEAR", Value: "LINEAR"},
										{Label: "CANARY", Value: "CANARY"},
									},
								},
							},
						},
						{
							Name:     "bakeTimeInMinutes",
							Label:    "Bake Time (minutes)",
							Type:     configuration.FieldTypeNumber,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Number: &configuration.NumberTypeOptions{
									Min: func() *int { min := 0; return &min }(),
								},
							},
						},
						{
							Name:     "canaryConfiguration",
							Label:    "Canary Configuration",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "canaryPercent",
											Label:    "Canary Percent",
											Type:     configuration.FieldTypeNumber,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												Number: &configuration.NumberTypeOptions{
													Min: func() *int { min := 0; return &min }(),
													Max: func() *int { max := 100; return &max }(),
												},
											},
										},
										{
											Name:     "canaryBakeTimeInMinutes",
											Label:    "Canary Bake Time (minutes)",
											Type:     configuration.FieldTypeNumber,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												Number: &configuration.NumberTypeOptions{
													Min: func() *int { min := 0; return &min }(),
												},
											},
										},
									},
								},
							},
						},
						{
							Name:     "linearConfiguration",
							Label:    "Linear Configuration",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "stepPercent",
											Label:    "Step Percent",
											Type:     configuration.FieldTypeNumber,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												Number: &configuration.NumberTypeOptions{
													Min: func() *int { min := 0; return &min }(),
													Max: func() *int { max := 100; return &max }(),
												},
											},
										},
										{
											Name:     "stepBakeTimeInMinutes",
											Label:    "Step Bake Time (minutes)",
											Type:     configuration.FieldTypeNumber,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												Number: &configuration.NumberTypeOptions{
													Min: func() *int { min := 0; return &min }(),
												},
											},
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
			Name:        "healthCheckGracePeriodSeconds",
			Label:       "Health Check Grace Period (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Grace period before ECS starts evaluating target health checks",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
				},
			},
		},
		{
			Name:        "serviceRegistries",
			Label:       "Service Registries",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Cloud Map service registry entries as ECS API objects",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Registry",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "registryArn",
								Label:    "Registry ARN",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "port",
								Label:    "Port",
								Type:     configuration.FieldTypeNumber,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 1; return &min }(),
									},
								},
							},
							{
								Name:     "containerName",
								Label:    "Container Name",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "containerPort",
								Label:    "Container Port",
								Type:     configuration.FieldTypeNumber,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 1; return &min }(),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "loadBalancers",
			Label:       "Load Balancers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Load balancer objects in ECS API format",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Load Balancer",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "targetGroupArn",
								Label:    "Target Group ARN",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "loadBalancerName",
								Label:    "Load Balancer Name",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "containerName",
								Label:    "Container Name",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "containerPort",
								Label:    "Container Port",
								Type:     configuration.FieldTypeNumber,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 1; return &min }(),
									},
								},
							},
							{
								Name:     "advancedConfiguration",
								Label:    "Advanced Configuration",
								Type:     configuration.FieldTypeObject,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Object: &configuration.ObjectTypeOptions{
										Schema: []configuration.Field{
											{
												Name:     "alternateTargetGroupArn",
												Label:    "Alternate Target Group ARN",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "productionListenerRule",
												Label:    "Production Listener Rule",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "testListenerRule",
												Label:    "Test Listener Rule",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "roleArn",
												Label:    "Role ARN",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
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
			Name:        "placementConstraints",
			Label:       "Placement Constraints",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Task placement constraint objects",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Constraint",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "distinctInstance", Value: "distinctInstance"},
											{Label: "memberOf", Value: "memberOf"},
										},
									},
								},
							},
							{
								Name:     "expression",
								Label:    "Expression",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
						},
					},
				},
			},
		},
		{
			Name:        "placementStrategy",
			Label:       "Placement Strategy",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Task placement strategy objects",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Strategy",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "random", Value: "random"},
											{Label: "spread", Value: "spread"},
											{Label: "binpack", Value: "binpack"},
										},
									},
								},
							},
							{
								Name:     "field",
								Label:    "Field",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
						},
					},
				},
			},
		},
		{
			Name:        "serviceConnectConfiguration",
			Label:       "Service Connect Configuration",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{}",
			Togglable:   true,
			Description: "ECS Service Connect configuration object",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "enabled",
							Label:    "Enabled",
							Type:     configuration.FieldTypeBool,
							Required: false,
						},
						{
							Name:     "namespace",
							Label:    "Namespace",
							Type:     configuration.FieldTypeString,
							Required: false,
						},
						{
							Name:     "services",
							Label:    "Services",
							Type:     configuration.FieldTypeList,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								List: &configuration.ListTypeOptions{
									ItemLabel: "Service",
									ItemDefinition: &configuration.ListItemDefinition{
										Type: configuration.FieldTypeObject,
										Schema: []configuration.Field{
											{
												Name:     "portName",
												Label:    "Port Name",
												Type:     configuration.FieldTypeString,
												Required: true,
											},
											{
												Name:     "discoveryName",
												Label:    "Discovery Name",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "ingressPortOverride",
												Label:    "Ingress Port Override",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 1; return &min }(),
													},
												},
											},
											{
												Name:     "clientAliases",
												Label:    "Client Aliases",
												Type:     configuration.FieldTypeList,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													List: &configuration.ListTypeOptions{
														ItemLabel: "Client Alias",
														ItemDefinition: &configuration.ListItemDefinition{
															Type: configuration.FieldTypeObject,
															Schema: []configuration.Field{
																{
																	Name:     "dnsName",
																	Label:    "DNS Name",
																	Type:     configuration.FieldTypeString,
																	Required: false,
																},
																{
																	Name:     "port",
																	Label:    "Port",
																	Type:     configuration.FieldTypeNumber,
																	Required: true,
																	TypeOptions: &configuration.TypeOptions{
																		Number: &configuration.NumberTypeOptions{
																			Min: func() *int { min := 1; return &min }(),
																		},
																	},
																},
															},
														},
													},
												},
											},
											{
												Name:     "timeout",
												Label:    "Timeout",
												Type:     configuration.FieldTypeObject,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Object: &configuration.ObjectTypeOptions{
														Schema: []configuration.Field{
															{
																Name:     "idleTimeoutSeconds",
																Label:    "Idle Timeout (seconds)",
																Type:     configuration.FieldTypeNumber,
																Required: false,
																TypeOptions: &configuration.TypeOptions{
																	Number: &configuration.NumberTypeOptions{
																		Min: func() *int { min := 0; return &min }(),
																	},
																},
															},
															{
																Name:     "perRequestTimeoutSeconds",
																Label:    "Per Request Timeout (seconds)",
																Type:     configuration.FieldTypeNumber,
																Required: false,
																TypeOptions: &configuration.TypeOptions{
																	Number: &configuration.NumberTypeOptions{
																		Min: func() *int { min := 0; return &min }(),
																	},
																},
															},
														},
													},
												},
											},
											{
												Name:     "tls",
												Label:    "TLS",
												Type:     configuration.FieldTypeObject,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Object: &configuration.ObjectTypeOptions{
														Schema: []configuration.Field{
															{
																Name:     "issuerCertificateAuthority",
																Label:    "Issuer Certificate Authority",
																Type:     configuration.FieldTypeObject,
																Required: false,
																TypeOptions: &configuration.TypeOptions{
																	Object: &configuration.ObjectTypeOptions{
																		Schema: []configuration.Field{
																			{
																				Name:     "awsPcaAuthorityArn",
																				Label:    "AWS PCA Authority ARN",
																				Type:     configuration.FieldTypeString,
																				Required: false,
																			},
																		},
																	},
																},
															},
															{
																Name:     "kmsKey",
																Label:    "KMS Key",
																Type:     configuration.FieldTypeString,
																Required: false,
															},
															{
																Name:     "roleArn",
																Label:    "Role ARN",
																Type:     configuration.FieldTypeString,
																Required: false,
															},
														},
													},
												},
											},
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
			Name:        "volumeConfigurations",
			Label:       "Volume Configurations",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Service-managed volume configuration objects",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Volume",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "managedEBSVolume",
								Label:    "Managed EBS Volume",
								Type:     configuration.FieldTypeObject,
								Required: false,
								TypeOptions: &configuration.TypeOptions{
									Object: &configuration.ObjectTypeOptions{
										Schema: []configuration.Field{
											{
												Name:     "encrypted",
												Label:    "Encrypted",
												Type:     configuration.FieldTypeBool,
												Required: false,
											},
											{
												Name:     "fileSystemType",
												Label:    "Filesystem Type",
												Type:     configuration.FieldTypeSelect,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Select: &configuration.SelectTypeOptions{
														Options: []configuration.FieldOption{
															{Label: "ext3", Value: "ext3"},
															{Label: "ext4", Value: "ext4"},
															{Label: "xfs", Value: "xfs"},
														},
													},
												},
											},
											{
												Name:     "iops",
												Label:    "IOPS",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
											{
												Name:     "kmsKeyId",
												Label:    "KMS Key ID",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "roleArn",
												Label:    "Role ARN",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "sizeInGiB",
												Label:    "Size (GiB)",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 1; return &min }(),
													},
												},
											},
											{
												Name:     "snapshotId",
												Label:    "Snapshot ID",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "tagSpecifications",
												Label:    "Tag Specifications",
												Type:     configuration.FieldTypeList,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													List: &configuration.ListTypeOptions{
														ItemLabel: "Tag Specification",
														ItemDefinition: &configuration.ListItemDefinition{
															Type: configuration.FieldTypeObject,
															Schema: []configuration.Field{
																{
																	Name:     "resourceType",
																	Label:    "Resource Type",
																	Type:     configuration.FieldTypeString,
																	Required: false,
																},
																{
																	Name:     "propagateTags",
																	Label:    "Propagate Tags",
																	Type:     configuration.FieldTypeSelect,
																	Required: false,
																	TypeOptions: &configuration.TypeOptions{
																		Select: &configuration.SelectTypeOptions{
																			Options: servicePropagateTagOptions,
																		},
																	},
																},
																{
																	Name:     "tags",
																	Label:    "Tags",
																	Type:     configuration.FieldTypeList,
																	Required: false,
																	TypeOptions: &configuration.TypeOptions{
																		List: &configuration.ListTypeOptions{
																			ItemLabel: "Tag",
																			ItemDefinition: &configuration.ListItemDefinition{
																				Type: configuration.FieldTypeObject,
																				Schema: []configuration.Field{
																					{
																						Name:     "key",
																						Label:    "Key",
																						Type:     configuration.FieldTypeString,
																						Required: true,
																					},
																					{
																						Name:     "value",
																						Label:    "Value",
																						Type:     configuration.FieldTypeString,
																						Required: true,
																					},
																				},
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
												Name:     "terminationPolicy",
												Label:    "Termination Policy",
												Type:     configuration.FieldTypeObject,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Object: &configuration.ObjectTypeOptions{
														Schema: []configuration.Field{
															{
																Name:     "deleteOnTermination",
																Label:    "Delete On Termination",
																Type:     configuration.FieldTypeBool,
																Required: false,
															},
														},
													},
												},
											},
											{
												Name:     "throughput",
												Label:    "Throughput",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
											{
												Name:     "volumeInitializationRate",
												Label:    "Volume Initialization Rate",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
											{
												Name:     "volumeType",
												Label:    "Volume Type",
												Type:     configuration.FieldTypeSelect,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Select: &configuration.SelectTypeOptions{
														Options: []configuration.FieldOption{
															{Label: "gp3", Value: "gp3"},
															{Label: "gp2", Value: "gp2"},
															{Label: "io1", Value: "io1"},
															{Label: "io2", Value: "io2"},
															{Label: "st1", Value: "st1"},
															{Label: "sc1", Value: "sc1"},
															{Label: "standard", Value: "standard"},
														},
													},
												},
											},
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
			Name:        "vpcLatticeConfigurations",
			Label:       "VPC Lattice Configurations",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "VPC Lattice configuration objects",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Configuration",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "portName",
								Label:    "Port Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "roleArn",
								Label:    "Role ARN",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "targetGroupArn",
								Label:    "Target Group ARN",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "availabilityZoneRebalancing",
			Label:       "Availability Zone Rebalancing",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Whether ECS should rebalance tasks across AZs",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: serviceAvailabilityZoneRebalancingOptions,
				},
			},
		},
		{
			Name:        "enableECSManagedTags",
			Label:       "Enable ECS Managed Tags",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Apply ECS managed tags to tasks launched by this service",
		},
		{
			Name:        "propagateTags",
			Label:       "Propagate Tags",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "Tag propagation source for tasks in this service",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: servicePropagateTagOptions,
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Service tags",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "key",
								Label:    "Key",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "additionalCreateOrUpdateArguments",
			Label:       "Additional ECS API Arguments",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{}",
			Togglable:   true,
			Description: "Additional key/value payload fields sent directly to ECS CreateService or UpdateService",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "deploymentController",
							Label:       "Deployment Controller",
							Type:        configuration.FieldTypeObject,
							Required:    false,
							Default:     "{\"type\":\"ECS\"}",
							Description: "Example advanced argument. Any other ECS API argument keys are also accepted.",
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "type",
											Label:    "Type",
											Type:     configuration.FieldTypeSelect,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												Select: &configuration.SelectTypeOptions{
													Options: []configuration.FieldOption{
														{Label: "ECS", Value: "ECS"},
														{Label: "CODE_DEPLOY", Value: "CODE_DEPLOY"},
														{Label: "EXTERNAL", Value: "EXTERNAL"},
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Name:     "availabilityZoneRebalancing",
							Label:    "Availability Zone Rebalancing",
							Type:     configuration.FieldTypeSelect,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: serviceAvailabilityZoneRebalancingOptions,
								},
							},
						},
					},
				},
			},
		},
	}

	if includeForceNewDeployment {
		fields = append(fields, configuration.Field{
			Name:        "forceNewDeployment",
			Label:       "Force New Deployment",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Start a new deployment even if the task definition has not changed",
		})
	}

	return fields
}
