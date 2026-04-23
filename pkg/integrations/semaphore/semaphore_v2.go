package semaphore

import (
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SemaphoreV2 struct{}

func (s *SemaphoreV2) Name() string {
	return "semaphore"
}

func (s *SemaphoreV2) Label() string {
	return "Semaphore"
}

func (s *SemaphoreV2) Description() string {
	return "Run and react to your Semaphore workflows"
}

func (s *SemaphoreV2) FirstStep() core.SetupStep {
	return core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         "selectOrganization",
		Instructions: "Select the organization you want to connect to",
		Inputs: []configuration.Field{
			{
				Name:     "organizationUrl",
				Label:    "Organization URL",
				Type:     configuration.FieldTypeString,
				Required: true,
			},
		},
	}
}

func (s *SemaphoreV2) OnStepSubmit(stepName string, input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch stepName {
	case "selectOrganization":
		return s.onSelectOrganizationSubmit(input, ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenSubmit(input, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SemaphoreV2) onSelectOrganizationSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	organizationURL, ok := m["organizationUrl"].(string)
	if !ok {
		return nil, errors.New("invalid organization URL")
	}

	if organizationURL == "" {
		return nil, errors.New("organization URL is required")
	}

	//
	// The organization URL is not something you can change,
	// so once it's set, you cannot change it.
	//
	err := ctx.Parameters.Create("organizationURL", core.IntegrationParameterDefinition{
		Type:     configuration.FieldTypeString,
		Value:    organizationURL,
		Editable: false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating configuration: %v", err)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         "enterAPIToken",
		Instructions: "Enter your Semaphore API token",
		Inputs: []configuration.Field{
			{
				Name:     "apiToken",
				Label:    "API Token",
				Type:     configuration.FieldTypeString,
				Required: true,
			},
		},
	}, nil
}

func (s *SemaphoreV2) onEnterAPITokenSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	apiToken, ok := m["apiToken"].(string)
	if !ok {
		return nil, errors.New("invalid API token")
	}

	if apiToken == "" {
		return nil, errors.New("API token is required")
	}

	client, err := NewClientV2(ctx.HTTP, ctx.Parameters, ctx.Secrets)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	//
	// Semaphore doesn't have a whoami endpoint, so
	// we list projects just to verify that the connection is working.
	//
	_, err = client.listProjects()
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %v", err)
	}

	//
	// API token is something you can change
	// so we store it as an editable secret.
	//
	err = ctx.Secrets.Create("apiToken", core.IntegrationSecretDefinition{
		Value:    []byte(apiToken),
		Editable: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	//
	// Register capabilities
	//
	err = ctx.Capabilities.RegisterComponents([]core.Component{
		&RunWorkflow{},
		&GetPipeline{},
	})

	if err != nil {
		return nil, fmt.Errorf("error registering components: %v", err)
	}

	err = ctx.Capabilities.RegisterTriggers([]core.Trigger{
		&OnPipelineDone{},
	})

	if err != nil {
		return nil, fmt.Errorf("error registering triggers: %v", err)
	}

	return nil, nil
}
