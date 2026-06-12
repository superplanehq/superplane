package claude

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

const (
	SetupStepCapabilitySelection = "capabilitySelection"
	SetupStepEnterAPIKey         = "enterAPIKey"
	SetupStepDone                = "done"
	SecretAPIKey                 = "apiKey"
)

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
	//
	// Group capabilities like Semaphore (see semaphore.SetupProvider.CapabilityGroups):
	// separate user-facing buckets so the picker scales when new actions are added.
	// — Messages & prompts: single-shot model calls (text, structured output, etc.).
	// — Agents: session-shaped or long-running work (Run Agent, future thread/webhook nodes).
	//
	return []core.CapabilityGroup{
		{
			Label: "Messages & prompts",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&TextPrompt{},
				},
				nil,
			),
		},
		{
			Label: "Agents",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&runagent.RunAgent{},
				},
				nil,
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
	case SetupStepEnterAPIKey:
		return s.onEnterAPIKeySubmit(ctx.Step.Inputs, ctx)
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         SetupStepEnterAPIKey,
		Label:        "Enter Claude API key",
		Instructions: (&Claude{}).Instructions(),
		Inputs: []configuration.Field{
			{
				Name:        SecretAPIKey,
				Label:       "API Key",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Claude API key",
			},
		},
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionRevert(ctx)
	case SetupStepEnterAPIKey:
		return s.onEnterAPIKeyRevert(ctx)
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

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for Claude")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case SecretAPIKey:
		v := strings.TrimSpace(ctx.Value)
		if v == "" {
			return nil, fmt.Errorf("value is required")
		}

		c := &Client{APIKey: v, BaseURL: defaultBaseURL, http: ctx.HTTP}
		if err := c.Verify(); err != nil {
			return nil, err
		}

		return nil, ctx.Secrets.Update(SecretAPIKey, v)

	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}
}

func (s *SetupProvider) onEnterAPIKeySubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
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

	c := &Client{APIKey: apiKey, BaseURL: defaultBaseURL, http: ctx.HTTP}
	if err := c.Verify(); err != nil {
		return nil, err
	}

	err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:        SecretAPIKey,
		Label:       "API Key",
		Description: "The Claude API key for this integration",
		Value:       apiKey,
		Editable:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         SetupStepDone,
		Label:        "Setup complete",
		Instructions: "Your Claude integration is ready. You can use the selected actions in workflows.",
	}, nil
}
