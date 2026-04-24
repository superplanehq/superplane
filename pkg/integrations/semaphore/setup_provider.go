package semaphore

import (
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SetupProvider struct{}

const organizationURLInstructions = `You can find the URL in the address bar of your browser when you are on the Semaphore organization page. It follows the format:

~~~text
https://<organization-name>.semaphoreci.com
~~~

For example, if your organization name is **superplane**, the URL would be:

~~~text
https://superplane.semaphoreci.com
~~~`

const apiTokenInstructions = `
There are two ways to provide a Semaphore API token:
1. Use a service account (recommended)
2. Use a personal API token

## 1. Use a service account (recommended)

If your organization has access to service accounts, you can use one of them to connect to SuperPlane.

1. Go to [%s/people](%s/people)
2. Create Service Account
   - Give it a name and a description
   - Give it an **Admin** role
3. Copy its API token and paste below

## 2. Use a personal API token

If your organization does not have access to service accounts, you can use a personal API token to connect to SuperPlane.

If you don't have access to your personal access token anymore, you can reset it:
1. Go to [%s](%s)
2. On the top right corner, click on your avatar and select **Profile Settings**
2. Reset API token, copy it and paste below

> **Important:**
> This will revoke the current token and generate a new one, so any existing workflows that use this token will stop working.
`

func (s *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	return core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  "selectOrganization",
		Label: "What is your Semaphore Organization URL?",
		Inputs: []configuration.Field{
			{
				Name:     "organizationUrl",
				Label:    "Semaphore Organization URL",
				Type:     configuration.FieldTypeString,
				Required: true,
				Default:  "https://hello.semaphoreci.com",
			},
		},
		Instructions: organizationURLInstructions,
	}
}

func (s *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.CurrentStep {
	case "selectOrganization":
		return s.onSelectOrganizationSubmit(ctx.Inputs, ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenSubmit(ctx.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.CurrentStep {
	case "selectOrganization":
		return s.onSelectOrganizationRevert(ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenRevert(ctx)
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) onSelectOrganizationRevert(ctx core.SetupStepContext) error {
	return ctx.Parameters.Delete("organizationUrl")
}

func (s *SetupProvider) onEnterAPITokenRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete("apiToken")
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
				Name:      "apiToken",
				Label:     "API Token",
				Type:      configuration.FieldTypeString,
				Required:  true,
				Sensitive: true,
			},
		},
		Instructions: fmt.Sprintf(apiTokenInstructions, organizationURL, organizationURL, organizationURL, organizationURL),
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
