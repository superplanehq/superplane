package railway

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
)

const (
	SetupStepCapabilitySelection = "capabilitySelection"
	SetupStepEnterAPIToken       = "enterAPIToken"
	SetupStepDone                = "done"
)

const (
	SecretAPIToken        = "apiToken"
	PropertyWorkspaceID   = "workspaceId"
	PropertyWorkspaceName = "workspaceName"
)

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
			Label: "Deployments",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&TriggerDeploy{},
				},
				[]core.Trigger{
					&OnDeploymentEvent{},
				},
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
	case SetupStepEnterAPIToken:
		return s.onEnterAPITokenSubmit(ctx.Step.Inputs, ctx)
	}
	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterAPIToken,
		Label: "Enter Railway Workspace Token",
		Inputs: []configuration.Field{
			{
				Name:      SecretAPIToken,
				Label:     "API Token",
				Type:      configuration.FieldTypeString,
				Required:  true,
				Sensitive: true,
			},
		},
		Instructions: string(apiTokenInstructionsTemplate),
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

	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return nil, errors.New("API token is required")
	}

	client := NewClientWithAPIToken(ctx.HTTP, apiToken)
	if err := client.Verify(); err != nil {
		return nil, fmt.Errorf("invalid API token: %w", err)
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workspaces: %w", err)
	}
	if len(workspaces) == 0 {
		return nil, errors.New("no workspaces accessible with this token")
	}

	workspace := workspaces[0]

	err = ctx.Properties.CreateMany([]core.IntegrationPropertyDefinition{
		{
			Name:        PropertyWorkspaceID,
			Label:       "Workspace ID",
			Description: "The ID of the connected Railway workspace",
			Type:        configuration.FieldTypeString,
			Value:       workspace.ID,
			Editable:    false,
		},
		{
			Name:        PropertyWorkspaceName,
			Label:       "Workspace Name",
			Description: "The name of the connected Railway workspace",
			Type:        configuration.FieldTypeString,
			Value:       workspace.Name,
			Editable:    false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save properties: %w", err)
	}

	err = ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:        SecretAPIToken,
		Label:       "API Token",
		Description: "Railway Workspace API Token",
		Value:       apiToken,
		Editable:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save secret: %w", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	tmpl, err := template.New("setupComplete").Parse(string(setupCompleteTemplate))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]any{
		"WorkspaceName": workspace.Name,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         SetupStepDone,
		Label:        "Setup complete",
		Instructions: buf.String(),
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case SetupStepEnterAPIToken:
		_ = ctx.Properties.Delete(PropertyWorkspaceID, PropertyWorkspaceName)
		_ = ctx.Secrets.Delete(SecretAPIToken)
		return nil
	}
	return errors.New("unknown step")
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, errors.New("property updates are not supported for Railway")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	switch ctx.SecretName {
	case SecretAPIToken:
		v := strings.TrimSpace(ctx.Value)
		if v == "" {
			return nil, errors.New("value is required")
		}

		client := NewClientWithAPIToken(ctx.HTTP, v)
		if err := client.Verify(); err != nil {
			return nil, fmt.Errorf("failed to verify new API token: %w", err)
		}

		return nil, ctx.Secrets.Update(SecretAPIToken, v)
	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}
}
