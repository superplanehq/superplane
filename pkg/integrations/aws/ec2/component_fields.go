package ec2

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func regionSelectField() configuration.Field {
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

func imageIDField(name, label, description, regionFieldName string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: description,
		Placeholder: "ami-1234567890abcdef0",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: "ec2.image",
				Parameters: []configuration.ParameterRef{
					{
						Name: "region",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: regionFieldName,
						},
					},
				},
			},
		},
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  regionFieldName,
				Values: []string{"*"},
			},
		},
	}
}
