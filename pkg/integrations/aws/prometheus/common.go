package prometheus

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const workspaceResourceType = "prometheus.workspace"

type workspaceConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	WorkspaceID string `json:"workspace" mapstructure:"workspace"`
	ClientToken string `json:"clientToken" mapstructure:"clientToken"`
}

type WorkspaceNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	WorkspaceID    string `json:"workspaceId" mapstructure:"workspaceId"`
	WorkspaceAlias string `json:"workspaceAlias" mapstructure:"workspaceAlias"`
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
		Description: "Tags to associate with the workspace",
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

func workspaceAliasFromExecution(ctx core.ExecutionContext) string {
	if ctx.NodeMetadata == nil {
		return ""
	}

	metadata := WorkspaceNodeMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return ""
	}

	return metadata.WorkspaceAlias
}

func noopWebhook() (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

type QueryOptionsConfiguration struct {
	Timeout                             string `json:"timeout" mapstructure:"timeout"`
	MaxSamplesProcessedWarningThreshold int    `json:"maxSamplesProcessedWarningThreshold" mapstructure:"maxSamplesProcessedWarningThreshold"`
	MaxSamplesProcessedErrorThreshold   int    `json:"maxSamplesProcessedErrorThreshold" mapstructure:"maxSamplesProcessedErrorThreshold"`
}

func queryField() configuration.Field {
	return configuration.Field{
		Name:        "query",
		Label:       "Query",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Placeholder: "up",
		Description: "PromQL expression to evaluate",
	}
}

func queryOptionFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional query timeout duration",
		},
		{
			Name:        "maxSamplesProcessedWarningThreshold",
			Label:       "Max Samples Warning Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Optional warning threshold for query samples processed",
		},
		{
			Name:        "maxSamplesProcessedErrorThreshold",
			Label:       "Max Samples Error Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Optional error threshold for query samples processed",
		},
	}
}

func validateQueryOptions(config QueryOptionsConfiguration) error {
	if config.MaxSamplesProcessedWarningThreshold < 0 {
		return fmt.Errorf("max samples warning threshold cannot be negative")
	}
	if config.MaxSamplesProcessedErrorThreshold < 0 {
		return fmt.Errorf("max samples error threshold cannot be negative")
	}

	return nil
}

func queryOutput(response map[string]any) map[string]any {
	data, ok := response["data"].(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return map[string]any{
		"resultType": data["resultType"],
		"result":     data["result"],
	}
}
