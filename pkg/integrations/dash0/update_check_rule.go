package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateCheckRule struct{}

type UpdateCheckRuleSpec struct {
	CheckRule     string          `mapstructure:"checkRule"`
	Name          string          `mapstructure:"name"`
	Expression    string          `mapstructure:"expression"`
	Dataset       string          `mapstructure:"dataset"`
	Thresholds    *ThresholdSpec  `mapstructure:"thresholds"`
	Summary       *string         `mapstructure:"summary"`
	Description   *string         `mapstructure:"description"`
	Interval      string          `mapstructure:"interval"`
	For           string          `mapstructure:"for"`
	KeepFiringFor string          `mapstructure:"keepFiringFor"`
	Labels        *[]KeyValuePair `mapstructure:"labels"`
	Annotations   *[]KeyValuePair `mapstructure:"annotations"`
	Enabled       bool            `mapstructure:"enabled"`
}

func (c *UpdateCheckRule) Name() string {
	return "dash0.updateCheckRule"
}

func (c *UpdateCheckRule) Label() string {
	return "Update Check Rule"
}

func (c *UpdateCheckRule) Description() string {
	return "Update an existing check rule (Prometheus alert rule) in Dash0"
}

func (c *UpdateCheckRule) Documentation() string {
	return `The Update Check Rule component updates an existing check rule (Prometheus alert rule) in Dash0. Use the check rule ID from a previous Create Check Rule output or from the Dash0 dashboard.

## Use Cases

- **Threshold adjustment**: Update alert thresholds based on changing conditions
- **Expression refinement**: Modify the PromQL query to better detect issues
- **Notification changes**: Update labels and annotations for better routing
- **Enable/disable**: Temporarily disable check rules during maintenance

## Configuration

- **Check Rule**: The Dash0 check rule ID to update (required)
- **Dataset**: The dataset the check rule belongs to (defaults to "default")
- **Name**, **Expression**, **Thresholds**, **Interval**, etc.: Same as Create Check Rule; the full spec is sent to replace the existing check rule

## Output

Returns the updated check rule details from the Dash0 API, including the rule ID and full configuration.`
}

func (c *UpdateCheckRule) Icon() string {
	return "bell"
}

func (c *UpdateCheckRule) Color() string {
	return "blue"
}

func (c *UpdateCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkRule",
			Label:       "Check Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The check rule to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "check-rule",
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable name for the check rule",
			Placeholder: "High error rate alert",
		},
		{
			Name:        "expression",
			Label:       "Expression",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "PromQL expression to evaluate. Use $__threshold for dynamic thresholds",
			Placeholder: "sum(rate(http_requests_total{status=~\"5..\"}[5m])) > $__threshold",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset to query",
		},
		{
			Name:        "thresholds",
			Label:       "Thresholds",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Threshold values for degraded and critical states. Required when using $__threshold in expression",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: checkRuleThresholdsSchema(),
				},
			},
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Short templatable summary (max 255 characters)",
			Placeholder: "Error rate is high",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Detailed templatable description (max 2048 characters)",
			Placeholder: "The error rate for service {{ $labels.service_name }} has exceeded the threshold",
		},
		{
			Name:     "interval",
			Label:    "Evaluation Interval",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "1m",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: checkRuleIntervalOptions(),
				},
			},
			Description: "How often to evaluate the expression",
		},
		{
			Name:     "for",
			Label:    "Trigger Grace Period",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "0",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: checkRuleGracePeriodOptions(),
				},
			},
			Description: "Multiplier of the evaluation interval the expression must be true before triggering",
		},
		{
			Name:     "keepFiringFor",
			Label:    "Keep Firing Grace Period",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "0",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: checkRuleGracePeriodOptions(),
				},
			},
			Description: "Multiplier of the evaluation interval to keep firing after expression becomes false",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Prometheus labels for routing and grouping",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: keyValueListSchema(),
					},
				},
			},
		},
		{
			Name:        "annotations",
			Label:       "Annotations",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Prometheus annotations for additional context",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Annotation",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: keyValueListSchema(),
					},
				},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    true,
			Default:     "true",
			Description: "Whether the check rule is active",
		},
	}
}

func (c *UpdateCheckRule) Setup(ctx core.SetupContext) error {
	spec := UpdateCheckRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CheckRule) == "" {
		return errors.New("checkRule is required")
	}

	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("name is required")
	}

	if strings.TrimSpace(spec.Expression) == "" {
		return errors.New("expression is required")
	}

	// If expression uses $__threshold, at least one threshold must be specified
	if strings.Contains(spec.Expression, "$__threshold") {
		if spec.Thresholds == nil || (spec.Thresholds.Degraded == nil && spec.Thresholds.Critical == nil) {
			return errors.New("at least one threshold (degraded or critical) is required when using $__threshold in expression")
		}
	}

	dataset := spec.Dataset
	if dataset == "" {
		dataset = "default"
	}
	err = resolveCheckRuleMetadata(ctx, spec.CheckRule, dataset)
	if err != nil {
		return fmt.Errorf("error resolving check rule metadata: %v", err)
	}

	return nil
}

func (c *UpdateCheckRule) Execute(ctx core.ExecutionContext) error {
	spec := UpdateCheckRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dataset := spec.Dataset
	if dataset == "" {
		dataset = "default"
	}

	// Build request from spec
	request := buildCheckRuleRequest(CreateCheckRuleSpec{
		Name:          spec.Name,
		Expression:    spec.Expression,
		Dataset:       dataset,
		Thresholds:    spec.Thresholds,
		Summary:       spec.Summary,
		Description:   spec.Description,
		Interval:      spec.Interval,
		For:           spec.For,
		KeepFiringFor: spec.KeepFiringFor,
		Labels:        spec.Labels,
		Annotations:   spec.Annotations,
		Enabled:       spec.Enabled,
	})

	data, err := client.UpdateCheckRule(spec.CheckRule, request, dataset)
	if err != nil {
		return fmt.Errorf("failed to update check rule: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.checkRule.updated",
		[]any{data},
	)
}

func (c *UpdateCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateCheckRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
