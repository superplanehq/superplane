package codebuild

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

type OnBuild struct{}

type OnBuildConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	Project string `json:"project" mapstructure:"project"`
}

type OnBuildMetadata struct {
	Region         string   `json:"region" mapstructure:"region"`
	SubscriptionID string   `json:"subscriptionId" mapstructure:"subscriptionId"`
	Project        *Project `json:"project" mapstructure:"project"`
}

func (p *OnBuild) Name() string {
	return "aws.codebuild.onBuild"
}

func (p *OnBuild) Label() string {
	return "CodeBuild • On Build"
}

func (p *OnBuild) Description() string {
	return "Listen to AWS CodeBuild build state change events"
}

func (p *OnBuild) Documentation() string {
	return `The On Build trigger starts a workflow execution when AWS CodeBuild emits a build state change event.

## Use Cases

- **Build observability**: Trigger notifications when builds fail, succeed, or stop
- **Automated quality gates**: Start follow-up checks after successful builds
- **Pipeline orchestration**: Chain downstream workflows based on build outcomes

## Event Data

Each build state change event includes:
- **detail.project-name**: CodeBuild project name
- **detail.build-status**: Build status (for example: IN_PROGRESS, SUCCEEDED, FAILED)
- **detail.build-id**: Build identifier ARN
- **detail.current-phase**: Current phase of the build lifecycle`
}

func (p *OnBuild) Icon() string {
	return "aws"
}

func (p *OnBuild) Color() string {
	return "gray"
}

func (p *OnBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
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
		},
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Filter by CodeBuild project name",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codebuild.project",
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
	}
}

func (p *OnBuild) Setup(ctx core.TriggerContext) error {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	config, err := decodeOnBuildConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	project, err := validateProject(ctx, config.Region, config.Project, metadata.Project)
	if err != nil {
		return fmt.Errorf("failed to validate project: %w", err)
	}

	//
	// If already subscribed for this project, nothing to do.
	//
	if metadata.SubscriptionID != "" && metadata.Project != nil && projectMatchesRef(metadata.Project, config.Project) {
		return nil
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	rule := common.EventBridgeRuleMetadata{}
	ok := false
	if integrationMetadata.EventBridge != nil && integrationMetadata.EventBridge.Rules != nil {
		rule, ok = integrationMetadata.EventBridge.Rules[Source]
	}
	if !ok || !slices.Contains(rule.DetailTypes, DetailTypeBuildStateChange) {
		if err := ctx.Metadata.Set(OnBuildMetadata{
			Region:  config.Region,
			Project: project,
		}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return p.provisionRule(ctx.Integration, ctx.Requests, config.Region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(config.Region, project.ProjectName))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnBuildMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
		Project:        project,
	})
}

func (p *OnBuild) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	err := integration.ScheduleActionCall(
		"provisionRule",
		common.ProvisionRuleParameters{
			Region:     region,
			Source:     Source,
			DetailType: DetailTypeBuildStateChange,
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

func (p *OnBuild) subscriptionPattern(region string, projectName string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeBuildStateChange,
		Source:     Source,
		Detail: map[string]any{
			"project-name": projectName,
		},
	}
}

func (p *OnBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if the EventBridge rule is available",
		},
	}
}

func (p *OnBuild) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return p.checkRuleAvailability(ctx)

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (p *OnBuild) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	rule := common.EventBridgeRuleMetadata{}
	ok := false
	if integrationMetadata.EventBridge != nil && integrationMetadata.EventBridge.Rules != nil {
		rule, ok = integrationMetadata.EventBridge.Rules[Source]
	}
	if !ok {
		ctx.Logger.Infof("Rule not found for source %s - checking again in 10 seconds", Source)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	if !slices.Contains(rule.DetailTypes, DetailTypeBuildStateChange) {
		ctx.Logger.Infof(
			"Rule does not have detail type '%s' - checking again in 10 seconds",
			DetailTypeBuildStateChange,
		)
		return nil, ctx.Requests.ScheduleActionCall(
			"checkRuleAvailability",
			map[string]any{},
			10*time.Second,
		)
	}

	if metadata.Project == nil || strings.TrimSpace(metadata.Project.ProjectName) == "" {
		return nil, fmt.Errorf("project metadata is missing")
	}

	subscriptionID, err := ctx.Integration.Subscribe(p.subscriptionPattern(metadata.Region, metadata.Project.ProjectName))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	return nil, ctx.Metadata.Set(metadata)
}

func (p *OnBuild) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	metadata := OnBuildMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	projectName, ok := event.Detail["project-name"]
	if !ok {
		return fmt.Errorf("missing project-name in event")
	}

	currentProject, ok := projectName.(string)
	if !ok || strings.TrimSpace(currentProject) == "" {
		return fmt.Errorf("invalid project-name in event")
	}

	if metadata.Project != nil && metadata.Project.ProjectName != "" && currentProject != metadata.Project.ProjectName {
		ctx.Logger.Infof("Skipping event for project %s, expected %s", currentProject, metadata.Project.ProjectName)
		return nil
	}

	return ctx.Events.Emit("aws.codebuild.build", ctx.Message)
}

func (p *OnBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (p *OnBuild) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func decodeOnBuildConfiguration(configuration any) (OnBuildConfiguration, error) {
	config := OnBuildConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnBuildConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Project = strings.TrimSpace(config.Project)

	if config.Region == "" {
		return OnBuildConfiguration{}, fmt.Errorf("region is required")
	}

	if config.Project == "" {
		return OnBuildConfiguration{}, fmt.Errorf("project is required")
	}

	return config, nil
}
