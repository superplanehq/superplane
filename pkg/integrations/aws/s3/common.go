package s3

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

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

func bucketField() configuration.Field {
	return configuration.Field{
		Name:        "bucket",
		Label:       "Bucket",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "Target S3 bucket",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: "s3.bucket",
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

func objectKeyField() configuration.Field {
	return configuration.Field{
		Name:        "key",
		Label:       "Object Key",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Description: "The key (path) of the object in the bucket",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "bucket",
				Values: []string{"*"},
			},
		},
	}
}
