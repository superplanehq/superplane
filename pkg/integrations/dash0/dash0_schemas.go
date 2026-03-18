package dash0

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CheckRuleNodeMetadata struct {
	CheckRule     string `json:"checkRule" mapstructure:"checkRule"`
	CheckRuleName string `json:"checkRuleName" mapstructure:"checkRuleName"`
}

func resolveCheckRuleMetadata(ctx core.SetupContext, checkRule, dataset string) error {
	// If the check rule is an expression placeholder, skip metadata resolution
	if strings.Contains(checkRule, "{{") {
		return nil
	}

	// If metadata is already set for the same check rule, skip the API call.
	var existing CheckRuleNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.CheckRule == checkRule && existing.CheckRuleName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %w", err)
	}

	data, err := client.GetCheckRule(checkRule, dataset)
	if err != nil {
		return fmt.Errorf("failed to get check rule during setup: %w", err)
	}

	var name string
	if spec, ok := data["spec"].(map[string]any); ok {
		name, _ = spec["name"].(string)
	}

	if name == "" {
		name, _ = data["name"].(string)
	}

	return ctx.Metadata.Set(CheckRuleNodeMetadata{
		CheckRule:     checkRule,
		CheckRuleName: name,
	})
}

// requestObjectSchema returns the request object fields for Create and Update HTTP synthetic check components.
func requestObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Target URL to monitor",
			Placeholder: "https://api.example.com/health",
		},
		{
			Name:     "method",
			Label:    "HTTP Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "get",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "get"},
						{Label: "POST", Value: "post"},
						{Label: "PUT", Value: "put"},
						{Label: "PATCH", Value: "patch"},
						{Label: "DELETE", Value: "delete"},
						{Label: "HEAD", Value: "head"},
					},
				},
			},
		},
		{
			Name:    "redirects",
			Label:   "Redirects",
			Type:    configuration.FieldTypeSelect,
			Default: "follow",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Follow", Value: "follow"},
						{Label: "Do not follow", Value: "do_not_follow"},
					},
				},
			},
			Description: "Whether to follow HTTP redirects",
		},
		{
			Name:    "allowInsecure",
			Label:   "Allow Insecure TLS",
			Type:    configuration.FieldTypeSelect,
			Default: "false",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No", Value: "false"},
						{Label: "Yes", Value: "true"},
					},
				},
			},
			Description: "Skip TLS certificate validation",
		},
		{
			Name:      "headers",
			Label:     "Headers",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true, Placeholder: "Content-Type"},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true, Placeholder: "application/json"},
						},
					},
				},
			},
			Description: "Custom HTTP request headers",
		},
		{
			Name:        "body",
			Label:       "Request Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Request body payload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"post", "put", "patch"}},
			},
		},
	}
}

// scheduleObjectSchema returns the schedule object fields for Create and Update HTTP synthetic check components.
func scheduleObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1m",
			Description: "How often the check runs (e.g. 30s, 1m, 5m, 1h, 2d)",
			Placeholder: "1m",
		},
		{
			Name:     "locations",
			Label:    "Locations",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"de-frankfurt"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Frankfurt (DE)", Value: "de-frankfurt"},
						{Label: "Oregon (US)", Value: "us-oregon"},
						{Label: "North Virginia (US)", Value: "us-north-virginia"},
						{Label: "London (UK)", Value: "uk-london"},
						{Label: "Brussels (BE)", Value: "be-brussels"},
						{Label: "Melbourne (AU)", Value: "au-melbourne"},
					},
				},
			},
			Description: "Locations to run the synthetic check from",
		},
		{
			Name:      "strategy",
			Label:     "Execution Strategy",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			Default:   "all_locations",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All locations", Value: "all_locations"},
						{Label: "Round-robin", Value: "round_robin"},
					},
				},
			},
			Description: "How checks are distributed across locations",
		},
	}
}

// retriesObjectSchema returns the retries object fields for Create and Update HTTP synthetic check components.
func retriesObjectSchema() []configuration.Field {
	return []configuration.Field{
		{Name: "attempts", Label: "Attempts", Type: configuration.FieldTypeNumber, Required: true, Default: "3", Description: "Number of retry attempts on failure"},
		{Name: "delay", Label: "Delay", Type: configuration.FieldTypeString, Required: true, Default: "1s", Description: "Delay between retries", Placeholder: "1s"},
	}
}

// checkRuleThresholdsSchema returns the thresholds object fields for check rules.
func checkRuleThresholdsSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "degraded",
			Label:       "Degraded Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   false,
			Description: "Threshold value for degraded state",
			Placeholder: "50",
		},
		{
			Name:        "critical",
			Label:       "Critical Threshold",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   false,
			Description: "Threshold value for critical state",
			Placeholder: "100",
		},
	}
}

// checkRuleIntervalOptions returns the evaluation interval options for check rules.
func checkRuleIntervalOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "1 minute", Value: "1m"},
		{Label: "2 minutes", Value: "2m"},
		{Label: "3 minutes", Value: "3m"},
		{Label: "4 minutes", Value: "4m"},
		{Label: "5 minutes", Value: "5m"},
		{Label: "10 minutes", Value: "10m"},
		{Label: "15 minutes", Value: "15m"},
		{Label: "1 hour", Value: "1h"},
	}
}

// checkRuleGracePeriodOptions returns the grace period multiplier options for check rules.
// The grace period is computed as multiplier × evaluation interval.
func checkRuleGracePeriodOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "None", Value: "0"},
		{Label: "1× evaluation interval", Value: "1"},
		{Label: "2× evaluation interval", Value: "2"},
		{Label: "3× evaluation interval", Value: "3"},
		{Label: "4× evaluation interval", Value: "4"},
		{Label: "5× evaluation interval", Value: "5"},
	}
}

// keyValueListSchema returns the list fields for labels/annotations (key-value pairs).
func keyValueListSchema() []configuration.Field {
	return []configuration.Field{
		{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true, Placeholder: "environment"},
		{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true, Placeholder: "production"},
	}
}
