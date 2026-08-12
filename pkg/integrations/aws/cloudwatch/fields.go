package cloudwatch

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// Configuration fields shared by the alarm components.

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

func unitField() configuration.Field {
	return configuration.Field{
		Name:        "unit",
		Label:       "Unit",
		Type:        configuration.FieldTypeSelect,
		Required:    false,
		Togglable:   true,
		Description: "Only needed when the metric is published with more than one unit",
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: AlarmUnitOptions,
			},
		},
	}
}

func alarmActionsField(name, label, description string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    false,
		Togglable:   true,
		Description: description,
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "region", Values: []string{"*"}},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:       "sns.topic",
				Multi:      true,
				Parameters: []configuration.ParameterRef{regionParameter()},
			},
		},
	}
}

func regionParameter() configuration.ParameterRef {
	return configuration.ParameterRef{
		Name:      "region",
		ValueFrom: &configuration.ParameterValueFrom{Field: "region"},
	}
}

func namespaceParameter() configuration.ParameterRef {
	return configuration.ParameterRef{
		Name:      "namespace",
		ValueFrom: &configuration.ParameterValueFrom{Field: "namespace"},
	}
}
