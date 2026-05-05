package openai

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
	"github.com/superplanehq/superplane/pkg/integrations/openai/common"
	"github.com/superplanehq/superplane/pkg/integrations/openai/components"
)

const (
	SetupStepCapabilitySelection = "capabilitySelection"
	SetupStepEnterAPIKey         = "enterAPIKey"
	SetupStepEnterBaseURL        = "enterBaseURL"
	SetupStepDone                = "done"
)

const (
	PropertyBaseURL = "baseURL"
	SecretAPIKey    = "apiKey"
)

const (
	apiKeyLabel       = "API Key"
	apiKeyDescription = "OpenAI API key"

	baseURLLabel       = "Base URL"
	baseURLDescription = "Custom API base URL for OpenAI-compatible providers"
)

//go:embed templates/api-key-instructions.tpl
var apiKeyInstructionsTemplate []byte

//go:embed templates/base-url-instructions.tpl
var baseURLInstructionsTemplate []byte

//go:embed templates/setup-complete.tpl
var setupCompleteTemplate []byte

type SetupProvider struct{}

func newSetupProvider() core.IntegrationSetupProvider {
	return &SetupProvider{}
}

func apiKeyField() configuration.Field {
	return configuration.Field{
		Name:        SecretAPIKey,
		Label:       apiKeyLabel,
		Type:        configuration.FieldTypeString,
		Required:    true,
		Sensitive:   true,
		Description: apiKeyDescription,
	}
}

func baseURLField() configuration.Field {
	return configuration.Field{
		Name:        PropertyBaseURL,
		Label:       baseURLLabel,
		Type:        configuration.FieldTypeString,
		Required:    false,
		Description: baseURLDescription,
		Placeholder: common.DefaultBaseURL,
	}
}

func apiKeySecret(value string) core.IntegrationSecretDefinition {
	return core.IntegrationSecretDefinition{
		Name:        SecretAPIKey,
		Label:       apiKeyLabel,
		Description: apiKeyDescription,
		Value:       value,
		Editable:    true,
	}
}

func baseURLProperty(value string) core.IntegrationPropertyDefinition {
	return core.IntegrationPropertyDefinition{
		Name:        PropertyBaseURL,
		Label:       baseURLLabel,
		Description: baseURLDescription,
		Type:        configuration.FieldTypeString,
		Value:       value,
		Editable:    true,
	}
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
					&components.CreateResponse{},
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
	return core.SetupStep{
		Type:         core.SetupStepTypeCapabilitySelection,
		Name:         SetupStepCapabilitySelection,
		Label:        "Select capabilities",
		Capabilities: s.allCapabilityNames(),
	}
}

func (s *SetupProvider) allCapabilityNames() []string {
	capabilities := []string{}
	for _, group := range s.CapabilityGroups() {
		for _, capability := range group.Capabilities {
			capabilities = append(capabilities, capability.Name)
		}
	}

	return capabilities
}

func (s *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionSubmit(ctx)
	case SetupStepEnterAPIKey:
		return s.onEnterAPIKeySubmit(ctx.Step.Inputs, ctx)
	case SetupStepEnterBaseURL:
		return s.onEnterBaseURLSubmit(ctx.Step.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterAPIKey,
		Label: "Enter OpenAI API key",
		Inputs: []configuration.Field{
			apiKeyField(),
		},
		Instructions: string(apiKeyInstructionsTemplate),
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionRevert(ctx)
	case SetupStepEnterAPIKey:
		return s.onEnterAPIKeyRevert(ctx)
	case SetupStepEnterBaseURL:
		return s.onEnterBaseURLRevert(ctx)
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionRevert(ctx core.SetupStepContext) error {
	ctx.Capabilities.Clear()
	return nil
}

func (s *SetupProvider) onEnterAPIKeyRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete(SecretAPIKey)
}

func (s *SetupProvider) onEnterBaseURLRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete(PropertyBaseURL)
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	switch ctx.PropertyName {
	case PropertyBaseURL:
		return s.onBaseURLUpdate(ctx)
	}

	return nil, fmt.Errorf("unknown property: %s", ctx.PropertyName)
}

func (s *SetupProvider) onBaseURLUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	apiKey, err := ctx.Secrets.Get(SecretAPIKey)
	if err != nil {
		return nil, fmt.Errorf("error getting API key: %v", err)
	}

	baseURL := strings.TrimSpace(ctx.Value)
	if err := common.NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
		return nil, err
	}

	if err := ctx.Properties.Delete(PropertyBaseURL); err != nil {
		return nil, err
	}

	return nil, ctx.Properties.Create(baseURLProperty(baseURL))
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case SecretAPIKey:
		return s.onAPIKeyUpdate(ctx)
	}

	return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
}

func (s *SetupProvider) onAPIKeyUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	apiKey := strings.TrimSpace(ctx.Value)
	if apiKey == "" {
		return nil, fmt.Errorf("value is required")
	}

	baseURL, _ := ctx.Properties.GetString(PropertyBaseURL)
	if err := common.NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
		return nil, err
	}

	return nil, ctx.Secrets.Update(SecretAPIKey, apiKey)
}

func (s *SetupProvider) onEnterAPIKeySubmit(inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	apiKey, err := parseAPIKeyInput(inputs)
	if err != nil {
		return nil, err
	}

	if err := ctx.Secrets.Create(apiKeySecret(apiKey)); err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	instructions, err := renderBaseURLInstructions()
	if err != nil {
		return nil, err
	}

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterBaseURL,
		Label: "Enter OpenAI Base URL",
		Inputs: []configuration.Field{
			baseURLField(),
		},
		Instructions: instructions,
	}, nil
}

func (s *SetupProvider) onEnterBaseURLSubmit(inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	baseURL, err := parseBaseURLInput(inputs)
	if err != nil {
		return nil, err
	}

	apiKey, err := ctx.Secrets.Get(SecretAPIKey)
	if err != nil {
		return nil, fmt.Errorf("error getting API key: %v", err)
	}

	if err := common.NewClientWithAPIKey(ctx.HTTP, apiKey, baseURL).Verify(); err != nil {
		return nil, err
	}

	if err := ctx.Properties.Create(baseURLProperty(baseURL)); err != nil {
		return nil, fmt.Errorf("error creating property: %v", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         SetupStepDone,
		Label:        "Setup complete",
		Instructions: string(setupCompleteTemplate),
	}, nil
}

func renderBaseURLInstructions() (string, error) {
	tmpl, err := template.New("baseURL").Parse(string(baseURLInstructionsTemplate))
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"DefaultBaseURL": common.DefaultBaseURL,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}

func parseAPIKeyInput(inputs any) (string, error) {
	m, ok := inputs.(map[string]any)
	if !ok {
		return "", errors.New("invalid input")
	}

	return requiredStringInput(m, SecretAPIKey, "invalid API key", "API key is required")
}

func parseBaseURLInput(inputs any) (string, error) {
	m, ok := inputs.(map[string]any)
	if !ok {
		return "", errors.New("invalid input")
	}

	return optionalStringInput(m, PropertyBaseURL, "invalid base URL")
}

func requiredStringInput(input map[string]any, name, invalidMessage, requiredMessage string) (string, error) {
	value, ok := input[name].(string)
	if !ok {
		return "", errors.New(invalidMessage)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New(requiredMessage)
	}

	return value, nil
}

func optionalStringInput(input map[string]any, name, invalidMessage string) (string, error) {
	raw, ok := input[name]
	if !ok {
		return "", nil
	}

	value, ok := raw.(string)
	if !ok {
		return "", errors.New(invalidMessage)
	}

	return strings.TrimSpace(value), nil
}
