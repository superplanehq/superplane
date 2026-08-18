package cloudwatch

import (
	"strings"

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

// unitField offers the metric units. Only Update Alarm can clear a unit, so
// only it gets the "No unit" option; on create, leaving the field off is the
// same thing.
func unitField(allowUnset bool) configuration.Field {
	options := AlarmUnitOptions
	if allowUnset {
		options = append([]configuration.FieldOption{AlarmUnitClearOption}, AlarmUnitOptions...)
	}

	return configuration.Field{
		Name:        "unit",
		Label:       "Unit",
		Type:        configuration.FieldTypeSelect,
		Required:    false,
		Togglable:   true,
		Description: "Only needed when the metric is published with more than one unit",
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: options,
			},
		},
	}
}

// resolveUnit maps the clear sentinel to the empty string the client omits.
func resolveUnit(value string) string {
	unit := strings.TrimSpace(value)
	if unit == UnitUnsetValue {
		return ""
	}

	return unit
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

// ec2ActionField exposes the EC2 automation action. Only Create Alarm can gate
// it on the namespace, since Update Alarm does not learn the alarm's namespace
// until it reads the alarm.
func ec2ActionField(visibilityConditions []configuration.VisibilityCondition) configuration.Field {
	return configuration.Field{
		Name:                 "ec2Action",
		Label:                "EC2 Action",
		Type:                 configuration.FieldTypeSelect,
		Required:             false,
		Togglable:            true,
		Description:          "EC2 automation CloudWatch runs when the alarm enters ALARM. Requires an AWS/EC2 alarm with an InstanceId dimension",
		VisibilityConditions: visibilityConditions,
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: AlarmEC2ActionOptions,
			},
		},
	}
}

// timestampTimezoneOptions is a curated set of IANA zones, matching the
// select statuspage.CreateIncident already offers for the same "disambiguate
// an offset-less timestamp" problem.
var timestampTimezoneOptions = []configuration.FieldOption{
	{Label: "UTC", Value: "UTC"},
	{Label: "America/New_York", Value: "America/New_York"},
	{Label: "America/Los_Angeles", Value: "America/Los_Angeles"},
	{Label: "America/Chicago", Value: "America/Chicago"},
	{Label: "Europe/London", Value: "Europe/London"},
	{Label: "Europe/Paris", Value: "Europe/Paris"},
	{Label: "Asia/Tokyo", Value: "Asia/Tokyo"},
	{Label: "Asia/Singapore", Value: "Asia/Singapore"},
}

// timestampTimezoneField lets a component disambiguate a timestamp field's
// offset-less values (e.g. from the datetime widget), which are otherwise
// ambiguously interpreted. Only relevant when the paired timestamp is set,
// so callers should gate its visibility on that field having a value.
func timestampTimezoneField(visibilityConditions []configuration.VisibilityCondition) configuration.Field {
	return configuration.Field{
		Name:                 "timezone",
		Label:                "Timezone",
		Type:                 configuration.FieldTypeSelect,
		Required:             false,
		Default:              "UTC",
		Description:          "Timezone the timestamp above is in, when it has no UTC offset of its own",
		VisibilityConditions: visibilityConditions,
		TypeOptions: &configuration.TypeOptions{
			Select: &configuration.SelectTypeOptions{
				Options: timestampTimezoneOptions,
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
