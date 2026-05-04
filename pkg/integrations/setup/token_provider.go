package setup

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	StepCapabilitySelection = "capabilitySelection"
	StepEnterCredentials    = "enterCredentials"
	StepDone                = "done"
)

type Property struct {
	Name        string
	Label       string
	Description string
	Default     string
	Placeholder string
	Required    bool
}

type Secret struct {
	Name        string
	Label       string
	Description string
}

type TokenProvider struct {
	IntegrationLabel       string
	CapabilityGroupLabel   string
	CredentialStepLabel    string
	CredentialInstructions string
	DoneInstructions       string
	Actions                []core.Action
	Triggers               []core.Trigger
	Properties             []Property
	Secrets                []Secret
	Validate               func(ctx core.SetupStepContext, values map[string]string) error
	ValidateSecret         func(ctx core.SecretUpdateContext, value string) error
}

func (p *TokenProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			Label:        p.CapabilityGroupLabel,
			Capabilities: p.capabilities(),
		},
	}
}

func (p *TokenProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	requested, ok := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if !ok {
		return nil, errors.New("no requested capabilities")
	}

	ctx.Capabilities.Enable(requested...)
	return nil, nil
}

func (p *TokenProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	capabilities := []string{}
	for _, capability := range p.capabilities() {
		capabilities = append(capabilities, capability.Name)
	}

	return core.SetupStep{
		Type:         core.SetupStepTypeCapabilitySelection,
		Name:         StepCapabilitySelection,
		Label:        "Select capabilities",
		Capabilities: capabilities,
	}
}

func (p *TokenProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step.Name {
	case StepCapabilitySelection:
		return p.onCapabilitySelectionSubmit(ctx)
	case StepEnterCredentials:
		return p.onCredentialsSubmit(ctx)
	default:
		return nil, errors.New("unknown step")
	}
}

func (p *TokenProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case StepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case StepEnterCredentials:
		for _, property := range p.Properties {
			if err := ctx.Properties.Delete(property.Name); err != nil {
				return err
			}
		}

		for _, secret := range p.Secrets {
			if err := ctx.Secrets.Delete(secret.Name); err != nil {
				return err
			}
		}

		return nil
	default:
		return errors.New("unknown step")
	}
}

func (p *TokenProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for %s", p.IntegrationLabel)
}

func (p *TokenProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	if !p.hasSecret(ctx.SecretName) {
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}

	value := strings.TrimSpace(ctx.Value)
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}

	if p.ValidateSecret != nil {
		if err := p.ValidateSecret(ctx, value); err != nil {
			return nil, err
		}
	}

	return nil, ctx.Secrets.Update(ctx.SecretName, value)
}

func (p *TokenProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(p.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         StepEnterCredentials,
		Label:        p.CredentialStepLabel,
		Inputs:       p.inputFields(),
		Instructions: p.CredentialInstructions,
	}, nil
}

func (p *TokenProvider) onCredentialsSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	inputs, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	values, err := p.extractValues(inputs)
	if err != nil {
		return nil, err
	}

	if p.Validate != nil {
		if err := p.Validate(ctx, values); err != nil {
			return nil, err
		}
	}

	if err := p.createProperties(ctx.Properties, values); err != nil {
		return nil, err
	}

	if err := p.createSecrets(ctx.Secrets, values); err != nil {
		return nil, err
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	instructions := p.DoneInstructions
	if instructions == "" {
		instructions = fmt.Sprintf("You are now connected to %s.", p.IntegrationLabel)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         StepDone,
		Label:        "Setup complete",
		Instructions: instructions,
	}, nil
}

func (p *TokenProvider) capabilities() []core.Capability {
	capabilities := []core.Capability{}
	for _, action := range p.Actions {
		capabilities = append(capabilities, core.Capability{
			Type:           core.IntegrationCapabilityTypeAction,
			Name:           action.Name(),
			Label:          action.Label(),
			Description:    action.Description(),
			Configuration:  action.Configuration(),
			OutputChannels: action.OutputChannels(nil),
		})
	}

	for _, trigger := range p.Triggers {
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

func (p *TokenProvider) capabilityDiff(capabilities []string) []string {
	diff := []string{}
	for _, capability := range p.capabilities() {
		if !slices.Contains(capabilities, capability.Name) {
			diff = append(diff, capability.Name)
		}
	}

	return diff
}

func (p *TokenProvider) inputFields() []configuration.Field {
	fields := []configuration.Field{}
	for _, property := range p.Properties {
		fields = append(fields, configuration.Field{
			Name:        property.Name,
			Label:       property.Label,
			Type:        configuration.FieldTypeString,
			Description: property.Description,
			Default:     property.Default,
			Placeholder: property.Placeholder,
			Required:    property.Required,
		})
	}

	for _, secret := range p.Secrets {
		fields = append(fields, configuration.Field{
			Name:        secret.Name,
			Label:       secret.Label,
			Type:        configuration.FieldTypeString,
			Description: secret.Description,
			Required:    true,
			Sensitive:   true,
		})
	}

	return fields
}

func (p *TokenProvider) extractValues(inputs map[string]any) (map[string]string, error) {
	values := map[string]string{}

	for _, property := range p.Properties {
		raw, ok := inputs[property.Name]
		if !ok {
			if property.Required {
				return nil, fmt.Errorf("%s is required", property.Label)
			}

			continue
		}

		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid %s", property.Label)
		}

		value = strings.TrimSpace(value)
		if property.Required && value == "" {
			return nil, fmt.Errorf("%s is required", property.Label)
		}

		if value != "" {
			values[property.Name] = value
		}
	}

	for _, secret := range p.Secrets {
		raw, ok := inputs[secret.Name]
		if !ok {
			return nil, fmt.Errorf("%s is required", secret.Label)
		}

		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid %s", secret.Label)
		}

		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s is required", secret.Label)
		}

		values[secret.Name] = value
	}

	return values, nil
}

func (p *TokenProvider) createProperties(storage core.IntegrationPropertyStorage, values map[string]string) error {
	for _, property := range p.Properties {
		value, ok := values[property.Name]
		if !ok {
			continue
		}

		if err := storage.Create(core.IntegrationPropertyDefinition{
			Name:        property.Name,
			Label:       property.Label,
			Description: property.Description,
			Type:        configuration.FieldTypeString,
			Value:       value,
			Editable:    false,
		}); err != nil {
			return fmt.Errorf("error creating property %s: %v", property.Name, err)
		}
	}

	return nil
}

func (p *TokenProvider) createSecrets(storage core.IntegrationSecretStorage, values map[string]string) error {
	for _, secret := range p.Secrets {
		if err := storage.Create(core.IntegrationSecretDefinition{
			Name:        secret.Name,
			Label:       secret.Label,
			Description: secret.Description,
			Value:       values[secret.Name],
			Editable:    true,
		}); err != nil {
			return fmt.Errorf("error creating secret %s: %v", secret.Name, err)
		}
	}

	return nil
}

func (p *TokenProvider) hasSecret(name string) bool {
	return slices.ContainsFunc(p.Secrets, func(secret Secret) bool {
		return secret.Name == name
	})
}
