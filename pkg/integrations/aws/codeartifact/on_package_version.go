package codeartifact

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	Source                              = "aws.codeartifact"
	DetailTypePackageVersionStateChange = "CodeArtifact Package Version State Change"
)

type OnPackageVersion struct{}

type OnPackageVersionConfiguration struct {
	Region     string                    `json:"region" mapstructure:"region"`
	Domain     string                    `json:"domain" mapstructure:"domain"`
	Repository string                    `json:"repository" mapstructure:"repository"`
	Packages   []configuration.Predicate `json:"packages" mapstructure:"packages"`
	Versions   []configuration.Predicate `json:"versions" mapstructure:"versions"`
}

type OnPackageVersionMetadata struct {
	Region         string      `json:"region" mapstructure:"region"`
	SubscriptionID string      `json:"subscriptionId" mapstructure:"subscriptionId"`
	Repository     *Repository `json:"repository" mapstructure:"repository"`
}

func (p *OnPackageVersion) Name() string {
	return "aws.codeArtifact.onPackageVersion"
}

func (p *OnPackageVersion) Label() string {
	return "CodeArtifact • On Package Version"
}

func (p *OnPackageVersion) Description() string {
	return "Listen to AWS CodeArtifact package version events"
}

func (p *OnPackageVersion) Documentation() string {
	return `The On Package Version trigger starts a workflow execution when a package version is created, modified, or deleted in AWS CodeArtifact.

## Use Cases

- **Release automation**: Trigger downstream workflows when a new package version is published
- **Dependency monitoring**: Notify teams about changes to shared libraries
- **Compliance checks**: Validate artifacts before promotion
`
}

func (p *OnPackageVersion) Icon() string {
	return "aws"
}

func (p *OnPackageVersion) Color() string {
	return "gray"
}

func (p *OnPackageVersion) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: RegionsForCodeArtifact,
				},
			},
		},
		{
			Name:     "domain",
			Label:    "Domain",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.domain",
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
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codeartifact.repository",
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
		},
		{
			Name:     "packages",
			Label:    "Packages",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": ".*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:     "versions",
			Label:    "Versions",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": ".*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnPackageVersion) Setup(ctx core.TriggerContext) error {
	metadata := OnPackageVersionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config := OnPackageVersionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	//
	// If already subscribed to the integration events, nothing to do.
	//
	if metadata.SubscriptionID != "" {
		return nil
	}

	repository, err := validateRepository(ctx.Integration, ctx.HTTP, region, config.Domain, config.Repository)
	if err != nil {
		return fmt.Errorf("failed to validate repository: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, region, DetailTypePackageVersionStateChange)
	if err != nil {
		return fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		if err := ctx.Metadata.Set(OnPackageVersionMetadata{Region: region, Repository: repository}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnPackageVersionMetadata{
		Region:         region,
		Repository:     repository,
		SubscriptionID: subscriptionID.String(),
	})
}

func (p *OnPackageVersion) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypePackageVersionStateChange,
		},
		time.Second,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule rule provisioning for integration: %w", err)
	}

	return requests.ScheduleActionCall(
		"checkRuleAvailability",
		map[string]any{},
		5*time.Second,
	)
}

func (p *OnPackageVersion) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypePackageVersionStateChange,
		Source:     Source,
		Detail: map[string]any{
			"operationType":       "Created",
			"packageVersionState": "Published",
		},
	}
}

func (p *OnPackageVersion) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnPackageVersion) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnPackageVersion) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnPackageVersionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, Source, metadata.Region, DetailTypePackageVersionStateChange)
	if err != nil {
		return nil, fmt.Errorf("failed to check rule availability: %w", err)
	}

	if !hasRule {
		return nil, ctx.Requests.ScheduleActionCall(ctx.Name, map[string]any{}, 10*time.Second)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnPackageVersion) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnPackageVersionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	fullPackageName, err := fullPackageName(event.Detail)
	if err != nil {
		return fmt.Errorf("failed to get full package name: %w", err)
	}

	if !configuration.MatchesAnyPredicate(config.Packages, fullPackageName) {
		ctx.Logger.Infof("Skipping event for package %s, does not match any predicate: %v", fullPackageName, config.Packages)
		return nil
	}

	version, ok := event.Detail["packageVersion"].(string)
	if !ok || version == "" {
		return fmt.Errorf("missing package version")
	}

	if !configuration.MatchesAnyPredicate(config.Versions, version) {
		ctx.Logger.Infof("Skipping event for version %s, does not match any predicate: %v", version, config.Versions)
		return nil
	}

	return ctx.Events.Emit("aws.codeartifact.package.version", ctx.Message)
}

func (p *OnPackageVersion) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (p *OnPackageVersion) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func fullPackageName(detail map[string]any) (string, error) {
	packageName, ok := detail["packageName"].(string)
	if !ok || packageName == "" {
		return "", fmt.Errorf("missing package name")
	}

	//
	// Package namespace can be empty.
	//
	packageNamespace, ok := detail["packageNamespace"].(string)
	if ok {
		return fmt.Sprintf("%s/%s", packageNamespace, packageName), nil
	}

	return packageName, nil
}
