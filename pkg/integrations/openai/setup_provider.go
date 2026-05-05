package openai

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	SetupStepCapabilitySelection = "capabilitySelection"
	SetupStepEnterCredentials    = "enterCredentials"
	SetupStepDone                = "done"
)

const (
	PropertyBaseURL = "baseURL"
	SecretAPIKey    = "apiKey"
)

type SetupProvider struct{}

func newSetupProvider() core.IntegrationSetupProvider {
	return &SetupProvider{}
}

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
			Label: "Models",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&CreateResponse{},
				},
				[]core.Trigger{},
			),
		},
	}
}

func (s *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
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
	case SetupStepEnterCredentials:
		return s.onEnterCredentialsSubmit(ctx.Step.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterCredentials,
		Label: "Enter OpenAI credentials",
		Inputs: []configuration.Field{
			{
				Name:        PropertyBaseURL,
				Label:       "Base URL",
				Type:        configuration.FieldTypeString,
				Required:    false,
				Description: "Custom API base URL for OpenAI-compatible providers",
				Placeholder: defaultBaseURL,
			},
			{
				Name:        SecretAPIKey,
				Label:       "API Key",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "OpenAI API key",
			},
		},
		Instructions: "Paste your OpenAI API key. Leave Base URL empty unless you use an OpenAI-compatible provider.",
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case SetupStepEnterCredentials:
		if err := ctx.Properties.Delete(PropertyBaseURL); err != nil {
			return err
		}

		return ctx.Secrets.Delete(SecretAPIKey)
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	switch ctx.PropertyName {
	case PropertyBaseURL:
		apiKey, err := ctx.Secrets.Get(SecretAPIKey)
		if err != nil {
			return nil, fmt.Errorf("error getting API key: %v", err)
		}

		baseURL := strings.TrimSpace(ctx.Value)
		if err := NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
			return nil, err
		}

		if err := ctx.Properties.Delete(PropertyBaseURL); err != nil {
			return nil, err
		}

		return nil, ctx.Properties.Create(core.IntegrationPropertyDefinition{
			Name:        PropertyBaseURL,
			Label:       "Base URL",
			Description: "Custom API base URL for OpenAI-compatible providers",
			Type:        configuration.FieldTypeString,
			Value:       baseURL,
			Editable:    true,
		})

	default:
		return nil, fmt.Errorf("unknown property: %s", ctx.PropertyName)
	}
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case SecretAPIKey:
		apiKey := strings.TrimSpace(ctx.Value)
		if apiKey == "" {
			return nil, fmt.Errorf("value is required")
		}

		baseURL, _ := ctx.Properties.GetString(PropertyBaseURL)
		if err := NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
			return nil, err
		}

		return nil, ctx.Secrets.Update(SecretAPIKey, apiKey)

	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}
}

func (s *SetupProvider) onEnterCredentialsSubmit(inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	apiKey, ok := m[SecretAPIKey].(string)
	if !ok {
		return nil, errors.New("invalid API key")
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	baseURL := ""
	if value, ok := m[PropertyBaseURL]; ok {
		baseURL, ok = value.(string)
		if !ok {
			return nil, errors.New("invalid base URL")
		}

		baseURL = strings.TrimSpace(baseURL)
	}

	if err := NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
		return nil, err
	}

	if err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:        PropertyBaseURL,
		Label:       "Base URL",
		Description: "Custom API base URL for OpenAI-compatible providers",
		Type:        configuration.FieldTypeString,
		Value:       baseURL,
		Editable:    true,
	}); err != nil {
		return nil, fmt.Errorf("error creating property: %v", err)
	}

	if err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:        SecretAPIKey,
		Label:       "API Key",
		Description: "OpenAI API key",
		Value:       apiKey,
		Editable:    true,
	}); err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         SetupStepDone,
		Label:        "Setup complete",
		Instructions: "OpenAI is connected.",
	}, nil
}
