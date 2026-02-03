package codeartifact

import (
	"fmt"
	"net/http"
	"slices"
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
	Region              string `json:"region" mapstructure:"region"`
	DomainName          string `json:"domainName" mapstructure:"domainName"`
	DomainOwner         string `json:"domainOwner" mapstructure:"domainOwner"`
	RepositoryName      string `json:"repositoryName" mapstructure:"repositoryName"`
	PackageFormat       string `json:"packageFormat" mapstructure:"packageFormat"`
	PackageNamespace    string `json:"packageNamespace" mapstructure:"packageNamespace"`
	PackageName         string `json:"packageName" mapstructure:"packageName"`
	PackageVersion      string `json:"packageVersion" mapstructure:"packageVersion"`
	PackageVersionState string `json:"packageVersionState" mapstructure:"packageVersionState"`
	OperationType       string `json:"operationType" mapstructure:"operationType"`
}

type OnPackageVersionMetadata struct {
	Region         string                        `json:"region" mapstructure:"region"`
	SubscriptionID string                        `json:"subscriptionId" mapstructure:"subscriptionId"`
	Filters        OnPackageVersionConfiguration `json:"filters" mapstructure:"filters"`
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

## Configuration

- **Region**: AWS region where the CodeArtifact domain lives
- **Domain Name**: Optional filter for the CodeArtifact domain
- **Repository Name**: Optional filter for the CodeArtifact repository
- **Package Format**: Optional filter for package format (e.g., ` + "`npm`" + `, ` + "`maven`" + `)
- **Package Name**: Optional filter for a specific package
- **Package Version State**: Optional filter for state (e.g., ` + "`Published`" + `)

## Event Data

Each event includes:
- **detail.domainName**: CodeArtifact domain name
- **detail.repositoryName**: Repository name
- **detail.packageName**: Package name
- **detail.packageVersion**: Package version
- **detail.packageVersionState**: Version state (Published, Disposed, etc.)
- **detail.operationType**: Operation (Created, Updated, Deleted)
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
			Type:     configuration.FieldTypeString,
			Required: true,
			Default:  "us-east-1",
		},
		{
			Name:        "domainName",
			Label:       "Domain Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by CodeArtifact domain",
		},
		{
			Name:        "domainOwner",
			Label:       "Domain Owner",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by CodeArtifact domain owner (AWS account ID)",
		},
		{
			Name:        "repositoryName",
			Label:       "Repository Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by CodeArtifact repository",
		},
		{
			Name:        "packageFormat",
			Label:       "Package Format",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by package format (e.g., npm, maven, pypi)",
		},
		{
			Name:        "packageNamespace",
			Label:       "Package Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by package namespace (for example npm scope)",
		},
		{
			Name:        "packageName",
			Label:       "Package Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by package name",
		},
		{
			Name:        "packageVersion",
			Label:       "Package Version",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by package version",
		},
		{
			Name:        "packageVersionState",
			Label:       "Package Version State",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by package version state (e.g., Published)",
		},
		{
			Name:        "operationType",
			Label:       "Operation Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by operation type (Created, Updated, Deleted)",
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

	config = normalizeConfiguration(config)
	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if metadata.SubscriptionID != "" && filtersEqual(metadata.Filters, config) {
		return nil
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	rule, ok := integrationMetadata.EventBridge.Rules[Source]
	if !ok || !slices.Contains(rule.DetailTypes, DetailTypePackageVersionStateChange) {
		err := ctx.Metadata.Set(OnPackageVersionMetadata{
			Region:  config.Region,
			Filters: config,
		})
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, config.Region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(config))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnPackageVersionMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
		Filters:        config,
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

func (p *OnPackageVersion) subscriptionPattern(config OnPackageVersionConfiguration) *common.EventBridgeEvent {
	detail := map[string]any{}
	if config.DomainName != "" {
		detail["domainName"] = config.DomainName
	}
	if config.DomainOwner != "" {
		detail["domainOwner"] = config.DomainOwner
	}
	if config.RepositoryName != "" {
		detail["repositoryName"] = config.RepositoryName
	}
	if config.PackageFormat != "" {
		detail["packageFormat"] = config.PackageFormat
	}
	if config.PackageNamespace != "" {
		detail["packageNamespace"] = config.PackageNamespace
	}
	if config.PackageName != "" {
		detail["packageName"] = config.PackageName
	}
	if config.PackageVersion != "" {
		detail["packageVersion"] = config.PackageVersion
	}
	if config.PackageVersionState != "" {
		detail["packageVersionState"] = config.PackageVersionState
	}
	if config.OperationType != "" {
		detail["operationType"] = config.OperationType
	}

	return &common.EventBridgeEvent{
		Region:     config.Region,
		DetailType: DetailTypePackageVersionStateChange,
		Source:     Source,
		Detail:     detail,
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

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	rule, ok := integrationMetadata.EventBridge.Rules[Source]
	if !ok {
		ctx.Logger.Infof("Rule not found for source %s - checking again in 10 seconds", Source)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	if !slices.Contains(rule.DetailTypes, DetailTypePackageVersionStateChange) {
		ctx.Logger.Infof("Rule does not have detail type '%s' - checking again in 10 seconds", DetailTypePackageVersionStateChange)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Filters))
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
	config = normalizeConfiguration(config)

	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if config.Region != "" && event.Region != "" && config.Region != event.Region {
		ctx.Logger.Infof("Skipping event for region %s, expected %s", event.Region, config.Region)
		return nil
	}

	if !matchesDetailFilters(event.Detail, config) {
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

func normalizeConfiguration(config OnPackageVersionConfiguration) OnPackageVersionConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.DomainName = strings.TrimSpace(config.DomainName)
	config.DomainOwner = strings.TrimSpace(config.DomainOwner)
	config.RepositoryName = strings.TrimSpace(config.RepositoryName)
	config.PackageFormat = strings.TrimSpace(config.PackageFormat)
	config.PackageNamespace = strings.TrimSpace(config.PackageNamespace)
	config.PackageName = strings.TrimSpace(config.PackageName)
	config.PackageVersion = strings.TrimSpace(config.PackageVersion)
	config.PackageVersionState = strings.TrimSpace(config.PackageVersionState)
	config.OperationType = strings.TrimSpace(config.OperationType)
	return config
}

func filtersEqual(a, b OnPackageVersionConfiguration) bool {
	return a.Region == b.Region &&
		a.DomainName == b.DomainName &&
		a.DomainOwner == b.DomainOwner &&
		a.RepositoryName == b.RepositoryName &&
		a.PackageFormat == b.PackageFormat &&
		a.PackageNamespace == b.PackageNamespace &&
		a.PackageName == b.PackageName &&
		a.PackageVersion == b.PackageVersion &&
		a.PackageVersionState == b.PackageVersionState &&
		a.OperationType == b.OperationType
}

func matchesDetailFilters(detail map[string]any, config OnPackageVersionConfiguration) bool {
	if len(detail) == 0 {
		return false
	}

	if !matchesDetailValue(detail, "domainName", config.DomainName) {
		return false
	}
	if !matchesDetailValue(detail, "domainOwner", config.DomainOwner) {
		return false
	}
	if !matchesDetailValue(detail, "repositoryName", config.RepositoryName) {
		return false
	}
	if !matchesDetailValue(detail, "packageFormat", config.PackageFormat) {
		return false
	}
	if !matchesDetailValue(detail, "packageNamespace", config.PackageNamespace) {
		return false
	}
	if !matchesDetailValue(detail, "packageName", config.PackageName) {
		return false
	}
	if !matchesDetailValue(detail, "packageVersion", config.PackageVersion) {
		return false
	}
	if !matchesDetailValue(detail, "packageVersionState", config.PackageVersionState) {
		return false
	}
	if !matchesDetailValue(detail, "operationType", config.OperationType) {
		return false
	}

	return true
}

func matchesDetailValue(detail map[string]any, key string, expected string) bool {
	if expected == "" {
		return true
	}

	value, ok := detail[key]
	if !ok || value == nil {
		return false
	}

	stringValue, ok := value.(string)
	if !ok {
		return false
	}

	return stringValue == expected
}
