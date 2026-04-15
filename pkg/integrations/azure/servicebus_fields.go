package azure

import "github.com/superplanehq/superplane/pkg/configuration"

// serviceBusResourceGroupField returns a resource group picker field for Service Bus components.
func serviceBusResourceGroupField() configuration.Field {
	return configuration.Field{
		Name:        "resourceGroup",
		Label:       "Resource Group",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Azure resource group containing the Service Bus namespace",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeResourceGroupDropdown,
				UseNameAsValue: true,
			},
		},
	}
}

// serviceBusNamespaceField returns a namespace picker field that cascades from resourceGroup.
func serviceBusNamespaceField() configuration.Field {
	return configuration.Field{
		Name:        "namespaceName",
		Label:       "Namespace",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Service Bus namespace",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusNamespace,
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "resourceGroup",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "resourceGroup",
						},
					},
				},
			},
		},
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "resourceGroup", Values: []string{"*"}},
		},
	}
}

// serviceBusNamespaceFieldStandalone returns a namespace picker field without a resourceGroup dependency.
// Used for data-plane-only components (Send Message, Publish Message) where resourceGroup is not needed.
func serviceBusNamespaceFieldStandalone() configuration.Field {
	return configuration.Field{
		Name:        "namespaceName",
		Label:       "Namespace",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Service Bus namespace",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusNamespace,
				UseNameAsValue: true,
			},
		},
	}
}

// serviceBusQueueNameField returns a queue picker that cascades from namespaceName.
func serviceBusQueueNameField() configuration.Field {
	return configuration.Field{
		Name:        "queueName",
		Label:       "Queue Name",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Service Bus queue",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusQueue,
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "namespaceName",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "namespaceName",
						},
					},
				},
			},
		},
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "namespaceName", Values: []string{"*"}},
		},
	}
}

// serviceBusTriggerNamespaceField returns a namespace picker for triggers where resourceGroup is
// optional. Unlike serviceBusNamespaceField it has no visibility condition, so it is always shown.
func serviceBusTriggerNamespaceField() configuration.Field {
	return configuration.Field{
		Name:        "namespaceName",
		Label:       "Namespace",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Service Bus namespace to monitor",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusNamespace,
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "resourceGroup",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "resourceGroup",
						},
					},
				},
			},
		},
	}
}

// serviceBusTriggerOptionalQueueNameField returns an optional queue picker that cascades from
// namespaceName. Becomes visible once a namespace is selected.
func serviceBusTriggerOptionalQueueNameField() configuration.Field {
	return configuration.Field{
		Name:        "queueName",
		Label:       "Queue Name",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    false,
		Description: "Filter to a specific queue (optional – leave empty for all queues in the namespace)",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusQueue,
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "namespaceName",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "namespaceName",
						},
					},
				},
			},
		},
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "namespaceName", Values: []string{"*"}},
		},
	}
}

// serviceBusTopicNameField returns a topic picker that cascades from namespaceName.
func serviceBusTopicNameField() configuration.Field {
	return configuration.Field{
		Name:        "topicName",
		Label:       "Topic Name",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The Service Bus topic",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeServiceBusTopic,
				UseNameAsValue: true,
				Parameters: []configuration.ParameterRef{
					{
						Name: "namespaceName",
						ValueFrom: &configuration.ParameterValueFrom{
							Field: "namespaceName",
						},
					},
				},
			},
		},
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "namespaceName", Values: []string{"*"}},
		},
	}
}
