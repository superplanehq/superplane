package semaphore

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/common"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/components"
)

const (
	SetupStepCapabilitySelection = "capabilitySelection"
	SetupStepSelectOrganization  = "selectOrganization"
	SetupStepEnterAPIToken       = "enterAPIToken"
	SetupStepDone                = "done"
)

const (
	PropertyOrganizationURL = "organizationUrl"
	SecretAPIToken          = "apiToken"
)

//go:embed templates/organization-url-instructions.tpl
var organizationURLInstructionsTemplate []byte

//go:embed templates/api-token-instructions.tpl
var apiTokenInstructionsTemplate []byte

//go:embed templates/setup-complete.tpl
var setupCompleteTemplate []byte

type SetupProvider struct{}

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
			ExampleOutput:  action.ExampleOutput(),
		})
	}
	for _, trigger := range triggers {
		capabilities = append(capabilities, core.Capability{
			Type:          core.IntegrationCapabilityTypeTrigger,
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Configuration: trigger.Configuration(),
			ExampleData:   trigger.ExampleData(),
		})
	}

	return capabilities
}

/*
 * Returns all the capabilities, minus the ones being passed in.
 */
func (s *SetupProvider) capabilityDiff(capabilities []string) []string {
	groups := s.CapabilityGroups()

	diff := []string{}
	for _, group := range groups {
		for _, capability := range group.Capabilities {
			if !slices.Contains(capabilities, capability.Name) {
				diff = append(diff, capability.Name)
			}
		}
	}

	return diff
}

func (s *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			Label: "Workflows",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&components.RunWorkflow{},
					&components.GetPipeline{},
				},
				[]core.Trigger{
					&components.OnPipelineDone{},
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

	ctx.Capabilities.Enable(requested...)
	return nil, nil
}

func (s *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	capabilities := []string{}
	for _, group := range s.CapabilityGroups() {
		for _, capability := range group.Capabilities {
			capabilities = append(capabilities, capability.Name)
		}
	}

	return core.SetupStep{
		Type:         core.SetupStepTypeCapabilitySelection,
		Name:         SetupStepCapabilitySelection,
		Label:        "Select capabilities",
		Capabilities: capabilities,
	}
}

func (s *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionSubmit(ctx)
	case SetupStepSelectOrganization:
		return s.onSelectOrganizationSubmit(ctx.Step.Inputs, ctx)
	case SetupStepEnterAPIToken:
		return s.onEnterAPITokenSubmit(ctx.Step.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {

	//
	// We move the requested capabilities to the REQUESTED state,
	// and the not requested ones to the AVAILABLE state,
	// since they were not requested yet, but they could be later on.
	//
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectOrganization,
		Label: "What is your Semaphore Organization URL?",
		Inputs: []configuration.Field{
			{
				Name:     PropertyOrganizationURL,
				Label:    "Semaphore Organization URL",
				Type:     configuration.FieldTypeString,
				Required: true,
				Default:  "https://hello.semaphoreci.com",
			},
		},
		Instructions: string(organizationURLInstructionsTemplate),
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionRevert(ctx)
	case SetupStepSelectOrganization:
		return s.onSelectOrganizationRevert(ctx)
	case SetupStepEnterAPIToken:
		return s.onEnterAPITokenRevert(ctx)
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionRevert(ctx core.SetupStepContext) error {
	ctx.Capabilities.Clear()
	return nil
}

func (s *SetupProvider) onSelectOrganizationRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete(PropertyOrganizationURL)
}

func (s *SetupProvider) onEnterAPITokenRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete(SecretAPIToken)
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for Semaphore")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case SecretAPIToken:
		v := strings.TrimSpace(ctx.Value)
		if v == "" {
			return nil, fmt.Errorf("value is required")
		}

		//
		// Validate the connection to Semaphore
		//
		client, err := common.NewClientWithAPIToken(ctx.HTTP, ctx.Properties, v)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}

		//
		// Semaphore doesn't have a whoami endpoint, so
		// we list projects just to verify that the connection is working.
		//
		_, err = client.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("error listing projects: %v", err)
		}

		return nil, ctx.Secrets.Update(SecretAPIToken, v)

	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}
}

func (s *SetupProvider) onSelectOrganizationSubmit(inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	organizationURL, ok := m[PropertyOrganizationURL].(string)
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
		Name:        PropertyOrganizationURL,
		Label:       "Organization URL",
		Description: "The URL of the Semaphore organization you are connected",
		Type:        configuration.FieldTypeString,
		Value:       organizationURL,
		Editable:    false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating configuration: %v", err)
	}

	tmpl, err := template.New("apiToken").Parse(string(apiTokenInstructionsTemplate))
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
		Name:  SetupStepEnterAPIToken,
		Label: "Enter Semaphore API token",
		Inputs: []configuration.Field{
			{
				Name:      SecretAPIToken,
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

	apiToken, ok := m[SecretAPIToken].(string)
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
		Name:        SecretAPIToken,
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
	client, err := common.NewClientWithStorageContexts(ctx.HTTP, ctx.Properties, ctx.Secrets)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	//
	// Semaphore doesn't have a whoami endpoint, so
	// we list projects just to verify that the connection is working.
	//
	projects, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
	url, err := ctx.Properties.GetString("organizationUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting organization URL: %v", err)
	}

	tmpl, err := template.New("setupCompleted").Parse(string(setupCompleteTemplate))
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
