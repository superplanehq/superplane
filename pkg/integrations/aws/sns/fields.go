package sns

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// regionField returns a reusable SNS region configuration field.
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

// topicField returns a reusable SNS topic selector field.
func topicField() configuration.Field {
	return configuration.Field{
		Name:        "topicArn",
		Label:       "Topic",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "ARN of the SNS topic",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: "sns.topic",
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

// subscriptionField returns a reusable SNS subscription selector field.
func subscriptionField() configuration.Field {
	return configuration.Field{
		Name:        "subscriptionArn",
		Label:       "Subscription",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "ARN of the SNS subscription",
		VisibilityConditions: []configuration.VisibilityCondition{
			{
				Field:  "region",
				Values: []string{"*"},
			},
		},
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: "sns.subscription",
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
