package semaphore

import (
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SetupProvider struct{}

func (s *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	return core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  "selectOrganization",
		Label: "Select Semaphore organization URL",
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

func (s *SetupProvider) OnStepSubmit(stepName string, inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch stepName {
	case "selectOrganization":
		return s.onSelectOrganizationSubmit(inputs, ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenSubmit(inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onSelectOrganizationSubmit(inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := inputs.(map[string]any)
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
	err := ctx.Parameters.Create(core.IntegrationParameterDefinition{
		Name:        "organizationUrl",
		Label:       "Organization URL",
		Description: "The URL of the Semaphore organization you are connected",
		Type:        configuration.FieldTypeString,
		Value:       organizationURL,
		Editable:    false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating configuration: %v", err)
	}

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  "enterAPIToken",
		Label: "Enter Semaphore API token",
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

func (s *SetupProvider) onEnterAPITokenSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
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

	//
	// API token is something you can change
	// so we store it as an editable secret.
	//
	err := ctx.Secrets.Create("apiToken", core.IntegrationSecretDefinition{
		Value:    []byte(apiToken),
		Editable: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	//
	// Validate the connection to Semaphore
	//
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
