package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateCheckRule struct{}

type CreateCheckRuleSpec struct {
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

type ThresholdSpec struct {
	Degraded *float64 `mapstructure:"degraded"`
	Critical *float64 `mapstructure:"critical"`
}

type KeyValuePair struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

func (c *CreateCheckRule) Name() string {
	return "dash0.createCheckRule"
}

func (c *CreateCheckRule) Label() string {
	return "Create Check Rule"
}

func (c *CreateCheckRule) Description() string {
	return "Create a Prometheus-style alert check rule in Dash0"
}

func (c *CreateCheckRule) Documentation() string {
	return `The Create Check Rule component creates a Prometheus-style alert check rule in Dash0 to monitor metrics and trigger alerts based on PromQL expressions.

## Use Cases

- **Service health monitoring**: Create alerts for service error rates, latency, or availability
- **Resource monitoring**: Alert on high CPU, memory, or disk usage
- **Business metrics**: Monitor key business metrics and trigger alerts when thresholds are exceeded
- **SLO enforcement**: Create alerts based on Service Level Objectives (SLOs)

## Configuration

### Name & Expression
- **Name**: Human-readable name for the check rule
- **Expression**: PromQL expression to evaluate. Supports $__threshold variable for dynamic thresholding

### Thresholds
- **Degraded**: Threshold value for degraded state (warning)
- **Critical**: Threshold value for critical state (alert)
- Required when using $__threshold in the expression

### Evaluation
- **Interval**: How often to evaluate the expression (1m, 5m, 10m)
- **For**: Grace period before triggering (pending duration)
- **Keep Firing For**: Grace period before resolving (resolution duration)

### Metadata
- **Summary**: Short templatable summary (max 255 chars)
- **Description**: Detailed templatable description (max 2048 chars)
- **Labels**: Prometheus labels for routing and grouping
- **Annotations**: Prometheus annotations for additional context

### Control
- **Enabled**: Whether the check rule is active
- **Dataset**: Dash0 dataset to query (defaults to "default")

## Output

Returns the created check rule details from the Dash0 API, including the rule ID and full configuration.`
}

func (c *CreateCheckRule) Icon() string {
	return "bell"
}

func (c *CreateCheckRule) Color() string {
	return "green"
}

func (c *CreateCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
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

func (c *CreateCheckRule) Setup(ctx core.SetupContext) error {
	spec := CreateCheckRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Expression == "" {
		return errors.New("expression is required")
	}

	// If expression uses $__threshold, at least one threshold must be specified
	if strings.Contains(spec.Expression, "$__threshold") {
		if spec.Thresholds == nil || (spec.Thresholds.Degraded == nil && spec.Thresholds.Critical == nil) {
			return errors.New("at least one threshold (degraded or critical) is required when using $__threshold in expression")
		}
	}

	return nil
}

func (c *CreateCheckRule) Execute(ctx core.ExecutionContext) error {
	spec := CreateCheckRuleSpec{}
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

	request := buildCheckRuleRequest(spec)

	data, err := client.CreateCheckRule(request, dataset)
	if err != nil {
		return fmt.Errorf("failed to create check rule: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.checkRule.created",
		[]any{data},
	)
}

func buildCheckRuleRequest(spec CreateCheckRuleSpec) CheckRuleRequest {
	interval := spec.Interval
	if interval == "" {
		interval = "1m"
	}

	forDuration := multiplyInterval(interval, spec.For)
	keepFiringFor := multiplyInterval(interval, spec.KeepFiringFor)

	request := CheckRuleRequest{
		Name:          spec.Name,
		Expression:    spec.Expression,
		Interval:      interval,
		For:           forDuration,
		KeepFiringFor: keepFiringFor,
		Enabled:       &spec.Enabled,
	}

	if spec.Thresholds != nil {
		request.Thresholds = &CheckRuleThresholds{
			Degraded: spec.Thresholds.Degraded,
			Critical: spec.Thresholds.Critical,
		}
	}

	if spec.Summary != nil {
		request.Summary = *spec.Summary
	}

	if spec.Description != nil {
		request.Description = *spec.Description
	}

	if spec.Labels != nil && len(*spec.Labels) > 0 {
		request.Labels = make(map[string]string)
		for _, label := range *spec.Labels {
			request.Labels[label.Key] = label.Value
		}
	}

	if spec.Annotations != nil && len(*spec.Annotations) > 0 {
		request.Annotations = make(map[string]string)
		for _, annotation := range *spec.Annotations {
			request.Annotations[annotation.Key] = annotation.Value
		}
	}

	return request
}

func multiplyInterval(interval string, multiplier string) string {
	if multiplier == "" || multiplier == "0" {
		return "0s"
	}

	m, err := strconv.Atoi(multiplier)
	if err != nil || m <= 0 {
		return "0s"
	}

	d, err := time.ParseDuration(interval)
	if err != nil {
		return "0s"
	}

	return formatDuration(d * time.Duration(m))
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}

	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}

	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func (c *CreateCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateCheckRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
