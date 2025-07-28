package semaphore

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/integrations"
)

type Hook struct {
	Workflow HookWorkflow
	Pipeline HookPipeline
}

type HookWorkflow struct {
	ID string `json:"id"`
}

type HookPipeline struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Result string `json:"result"`
}

func (i *SemaphoreIntegration) SetupWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	//
	// Create Semaphore secret to store the event source key.
	//
	resourceName := fmt.Sprintf("superplane-webhook-%s", options.ID)
	secret, err := i.createSemaphoreSecret(resourceName, options.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore secret: %v", err)
	}

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := i.createSemaphoreNotification(resourceName, options)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore notification: %v", err)
	}

	return []integrations.Resource{secret, notification}, nil
}

func (i *SemaphoreIntegration) createSemaphoreSecret(name string, key []byte) (integrations.Resource, error) {
	//
	// Check if secret already exists.
	//
	secret, err := i.Get(ResourceTypeSecret, name)
	if err == nil {
		return secret, nil
	}

	//
	// Secret does not exist, create it.
	//
	secret, err = i.Create(ResourceTypeSecret, &Secret{
		Metadata: SecretMetadata{
			Name: name,
		},
		Data: SecretSpecData{
			EnvVars: []SecretSpecDataEnvVar{
				{
					Name:  "WEBHOOK_SECRET",
					Value: string(key),
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	return secret, nil
}

func (i *SemaphoreIntegration) createSemaphoreNotification(name string, options integrations.WebhookOptions) (integrations.Resource, error) {
	//
	// Check if notification already exists.
	//
	notification, err := i.Get(ResourceTypeNotification, name)
	if err == nil {
		return notification, nil
	}

	//
	// Notification does not exist, create it.
	//
	notification, err = i.Create(ResourceTypeNotification, &Notification{
		Metadata: NotificationMetadata{
			Name: name,
		},
		Spec: NotificationSpec{
			Rules: []NotificationRule{
				{
					Name: fmt.Sprintf("webhook-for-%s", options.Resource.Name()),
					Filter: NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{options.Resource.Name()},
						Results:   []string{},
					},
					Notify: NotificationRuleNotify{
						Webhook: NotificationNotifyWebhook{
							Endpoint: options.URL,
							Secret:   name,
						},
					},
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating notification: %v", err)
	}

	return notification, nil
}

func (i *SemaphoreIntegration) HandleWebhook(data []byte) (integrations.StatefulResource, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling webhook data: %v", err)
	}

	return &Workflow{
		WfID: hook.Workflow.ID,
		Pipeline: &Pipeline{
			PipelineID: hook.Pipeline.ID,
			State:      hook.Pipeline.State,
			Result:     hook.Pipeline.Result,
		},
	}, nil
}
