package prometheus

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	workspaceResourceType           = "prometheus.workspace"
	ruleGroupsNamespaceResourceType = "prometheus.ruleGroupNamespace"
)

type workspaceConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	WorkspaceID string `json:"workspace" mapstructure:"workspace"`
	ClientToken string `json:"clientToken" mapstructure:"clientToken"`
}

type ruleGroupsNamespaceConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	WorkspaceID string `json:"workspace" mapstructure:"workspace"`
	Name        string `json:"namespace" mapstructure:"namespace"`
	ClientToken string `json:"clientToken" mapstructure:"clientToken"`
}

type WorkspaceNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	WorkspaceID    string `json:"workspaceId" mapstructure:"workspaceId"`
	WorkspaceAlias string `json:"workspaceAlias" mapstructure:"workspaceAlias"`
}

type RuleGroupsNamespaceNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	WorkspaceID    string `json:"workspaceId" mapstructure:"workspaceId"`
	WorkspaceAlias string `json:"workspaceAlias" mapstructure:"workspaceAlias"`
	Namespace      string `json:"namespace" mapstructure:"namespace"`
}

func regionField() configuration.Field {
	return configuration.Field{
		Name:     "region",
		Label:    "Region",
		Type:     configuration.FieldTypeSelect,
		Required: true,
		Default:  "us-east-1",
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: common.AllRegions,
			},
		},
	}
}

func workspaceField(label string, description string) configuration.Field {
	return configuration.Field{
		Name:        "workspace",
		Label:       label,
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: description,
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: workspaceResourceType,
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

func ruleGroupsNamespaceField(label string, description string) configuration.Field {
	return configuration.Field{
		Name:        "namespace",
		Label:       label,
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: description,
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "workspace",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: ruleGroupsNamespaceResourceType,
				Parameters: []configuration.ParameterRef{
					{
						Name: "region",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "region",
						},
					},
					{
						Name: "workspace",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "workspace",
						},
					},
				},
			},
		},
	}
}

func ruleGroupsNamespaceNameField() configuration.Field {
	return configuration.Field{
		Name:        "name",
		Label:       "Namespace Name",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Description: "Name for the rule group namespace",
		TypeOptions: &configuration.TypeOptions{
			String: &configuration.StringTypeOptions{
				MaxLength: func() *int { max := 128; return &max }(),
			},
		},
	}
}

func ruleGroupsNamespaceDataField() configuration.Field {
	allowExpressions := false

	return configuration.Field{
		Name:        "data",
		Label:       "Rule Groups YAML",
		Type:        configuration.FieldTypeText,
		Required:    true,
		Description: "Prometheus rule groups YAML for the namespace",
		TypeOptions: &configuration.TypeOptions{
			Text: &configuration.TextTypeOptions{
				Language:         "yaml",
				AllowExpressions: &allowExpressions,
			},
		},
	}
}

func ruleGroupsNamespaceOutput(namespace *RuleGroupsNamespaceSummary) map[string]any {
	if namespace == nil {
		return nil
	}

	output := map[string]any{
		"arn":  namespace.Arn,
		"name": namespace.Name,
	}

	if len(namespace.Tags) > 0 {
		output["tags"] = namespace.Tags
	}

	return output
}

func aliasField(required bool, description string) configuration.Field {
	return configuration.Field{
		Name:        "alias",
		Label:       "Alias",
		Type:        configuration.FieldTypeString,
		Required:    required,
		Description: description,
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			String: &configuration.StringTypeOptions{
				MaxLength: func() *int { max := 100; return &max }(),
			},
		},
	}
}

func clientTokenField() configuration.Field {
	return configuration.Field{
		Name:        "clientToken",
		Label:       "Client Token",
		Type:        configuration.FieldTypeString,
		Required:    false,
		Togglable:   true,
		Description: "Optional idempotency token",
		TypeOptions: &configuration.TypeOptions{
			String: &configuration.StringTypeOptions{
				MaxLength: func() *int { max := 64; return &max }(),
			},
		},
	}
}

func tagsField() configuration.Field {
	return configuration.Field{
		Name:        "tags",
		Label:       "Tags",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: "Tags to associate with the resource",
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
							Required: false,
						},
					},
				},
			},
		},
	}
}

func decodeWorkspaceConfiguration(rawConfiguration any) (workspaceConfiguration, error) {
	config := workspaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return workspaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.ClientToken = strings.TrimSpace(config.ClientToken)

	if config.Region == "" {
		return workspaceConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return workspaceConfiguration{}, fmt.Errorf("workspace is required")
	}

	return config, nil
}

func decodeRuleGroupsNamespaceConfiguration(rawConfiguration any) (ruleGroupsNamespaceConfiguration, error) {
	config := ruleGroupsNamespaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return ruleGroupsNamespaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Name = strings.TrimSpace(config.Name)
	config.ClientToken = strings.TrimSpace(config.ClientToken)

	if config.Region == "" {
		return ruleGroupsNamespaceConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return ruleGroupsNamespaceConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Name == "" {
		return ruleGroupsNamespaceConfiguration{}, fmt.Errorf("namespace is required")
	}

	return config, nil
}

func workspaceClient(ctx core.ExecutionContext, region string) (*Client, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	return NewClient(ctx.HTTP, creds, region), nil
}

func workspaceSetupClient(ctx core.SetupContext, region string) (*Client, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	return NewClient(ctx.HTTP, creds, region), nil
}

func resolveWorkspaceNodeMetadata(ctx core.SetupContext, config workspaceConfiguration) WorkspaceNodeMetadata {
	metadata := WorkspaceNodeMetadata{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}

	metadata.WorkspaceAlias = resolveWorkspaceAlias(ctx, config.Region, config.WorkspaceID)
	return metadata
}

func setWorkspaceNodeMetadata(ctx core.SetupContext, metadata WorkspaceNodeMetadata) error {
	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(metadata)
}

func resolveRuleGroupsNamespaceNodeMetadata(
	ctx core.SetupContext,
	config ruleGroupsNamespaceConfiguration,
) RuleGroupsNamespaceNodeMetadata {
	workspace := workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}

	return RuleGroupsNamespaceNodeMetadata{
		Region:         config.Region,
		WorkspaceID:    config.WorkspaceID,
		WorkspaceAlias: resolveWorkspaceNodeMetadata(ctx, workspace).WorkspaceAlias,
		Namespace:      config.Name,
	}
}

func setRuleGroupsNamespaceNodeMetadata(ctx core.SetupContext, metadata RuleGroupsNamespaceNodeMetadata) error {
	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(metadata)
}

func resolveWorkspaceAlias(ctx core.SetupContext, region string, workspaceID string) string {
	if ctx.HTTP == nil || ctx.Integration == nil || region == "" || workspaceID == "" {
		return workspaceID
	}

	client, err := workspaceSetupClient(ctx, region)
	if err != nil {
		return workspaceID
	}

	workspace, err := client.DescribeWorkspace(workspaceID)
	if err != nil {
		return workspaceID
	}

	alias := strings.TrimSpace(workspace.Alias)
	if alias == "" {
		return workspaceID
	}

	return alias
}

func noopWebhook() (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
