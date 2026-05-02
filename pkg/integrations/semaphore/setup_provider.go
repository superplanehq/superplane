package semaphore

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

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

const apiTokenTemplate = `
There are two ways to provide a Semaphore API token:
1. Use a service account - **recommended**
2. Use a personal API token
---
## 1. Use a service account
If your organization has access to service accounts, you can use one of them to connect to SuperPlane.
- Go to {{ .OrganizationURL }}/people
- Create a service account, with the **Admin** role
- Copy its API token and paste below
---
## 2. Use a personal API token
If your organization does not have access to service accounts, you can use a personal API token to connect to SuperPlane:
- Go to {{ .OrganizationURL }}
- On the top right corner, click on your avatar and select **Profile Settings**
- Reset the API token, copy it and paste below
> **Warning:**
> This will revoke the current token and generate a new one, so any existing workflows that use this token will stop working.
`

const setupCompletedTemplate = `
{{- $organizationURL := .OrganizationURL }}
You are now connected to {{ $organizationURL }}
---
You can now start using the following projects:
| Project | Repository |
|---------|------------|
{{- range .Projects }}
| [{{ .Metadata.ProjectName }}]({{ $organizationURL }}/projects/{{ .Metadata.ProjectName }}) | ` + "`{{ .Spec.Repository.URL }}`" + ` |
{{- end }}
`

func (s *SetupProvider) genCapabilities(actions []core.Action, triggers []core.Trigger) []core.Capability {
	capabilities := []core.Capability{}
	for _, action := range actions {
		capabilities = append(capabilities, core.Capability{
			Type:           core.IntegrationCapabilityTypeAction,
			Name:           action.Name(),
			Label:          action.Label(),
			Description:    action.Description(),
			Configuration:  action.Configuration(),
			OutputChannels: action.OutputChannels(nil),
		})
	}
	for _, trigger := range triggers {
		capabilities = append(capabilities, core.Capability{
			Type:          core.IntegrationCapabilityTypeTrigger,
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Configuration: trigger.Configuration(),
		})
	}

	return capabilities
}

func (s *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			Label: "All",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&RunWorkflow{},
					&GetPipeline{},
				},
				[]core.Trigger{
					&OnPipelineDone{},
				},
			),
		},
	}
}

func (s *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	//
	// The token we have already has permissions to do all the capabilities that Semaphore offers.
	// The only thing to do here is to enable the newly requested capabilities.
	//
	requested, ok := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if !ok {
		return nil, errors.New("no requested capabilities")
	}

	ctx.Logger.Infof("requested capabilities: %v", requested)
	return nil, ctx.Capabilities.Enable(requested...)
}

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
	switch ctx.Step {
	case "selectOrganization":
		return s.onSelectOrganizationSubmit(ctx.Inputs, ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenSubmit(ctx.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step {
	case "selectOrganization":
		return s.onSelectOrganizationRevert(ctx)
	case "enterAPIToken":
		return s.onEnterAPITokenRevert(ctx)
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) onSelectOrganizationRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete("organizationUrl")
}

func (s *SetupProvider) onEnterAPITokenRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete("apiToken")
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for Semaphore")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case "apiToken":
		v := strings.TrimSpace(ctx.Value)
		if v == "" {
			return nil, fmt.Errorf("value is required")
		}

		//
		// Validate the connection to Semaphore
		//
		client, err := NewClientWithAPIToken(ctx.HTTP, ctx.Properties, v)
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

		return nil, ctx.Secrets.Update("apiToken", v)

	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}
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
	err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
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

	tmpl, err := template.New("apiToken").Parse(apiTokenTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"OrganizationURL": organizationURL,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
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
		Instructions: buf.String(),
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
	err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:        "apiToken",
		Label:       "API Token",
		Description: "The API token for the Semaphore organization",
		Value:       apiToken,
		Editable:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	//
	// Validate the connection to Semaphore
	//
	client, err := NewClientWithStorageContexts(ctx.HTTP, ctx.Properties, ctx.Secrets)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	//
	// Semaphore doesn't have a whoami endpoint, so
	// we list projects just to verify that the connection is working.
	//
	projects, err := client.listProjects()
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %v", err)
	}

	err = ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
	if err != nil {
		return nil, fmt.Errorf("error enabling capability group: %v", err)
	}

	url, err := ctx.Properties.GetString("organizationUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting organization URL: %v", err)
	}

	tmpl, err := template.New("setupCompleted").Parse(setupCompletedTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"OrganizationURL": url,
		"Projects":        projects,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         "done",
		Label:        "Setup complete",
		Instructions: buf.String(),
	}, nil
}
