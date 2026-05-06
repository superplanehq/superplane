package cursor

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
	SetupStepEnterKeys           = "enterKeys"
	SetupStepDone                = "done"
)

const (
	SecretLaunchAgentKey = "launchAgentKey"
	SecretAdminKey       = "adminKey"
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
			// Things that create/manage Cursor Cloud Agents.
			// Future-friendly examples: list agents, cancel agent, get agent status, get conversation, etc.
			Label: "Agents",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&LaunchAgent{},
					&GetLastMessage{},
				},
				nil,
			),
		},
		{
			// Admin/teams endpoints and analytics.
			// Future-friendly examples: usage summaries by user/model, billing exports, seat counts, etc.
			Label: "Admin & Usage",
			Capabilities: s.genCapabilities(
				[]core.Action{
					&GetDailyUsageData{},
				},
				nil,
			),
		},
	}
}

func capabilityNeedsLaunchAgentKey(name string) bool {
	return name == (&LaunchAgent{}).Name() || name == (&GetLastMessage{}).Name()
}

func capabilityNeedsAdminKey(name string) bool {
	return name == (&GetDailyUsageData{}).Name()
}

func requestedNeedsLaunchAgentKey(requested []string) bool {
	return slices.ContainsFunc(requested, capabilityNeedsLaunchAgentKey)
}

func requestedNeedsAdminKey(requested []string) bool {
	return slices.ContainsFunc(requested, capabilityNeedsAdminKey)
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
	case SetupStepEnterKeys:
		return s.onEnterKeysSubmit(ctx.Step.Inputs, ctx)
	default:
		return nil, errors.New("unknown step")
	}
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case SetupStepEnterKeys:
		// Do not delete stored secrets here. Going "back" from the done step should not
		// wipe keys the user already had (e.g. expansion flows or pre-existing secrets).
		// Users can rotate keys via integration secret update.
		return nil
	default:
		return errors.New("unknown step")
	}
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for Cursor")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	v := strings.TrimSpace(ctx.Value)
	if v == "" {
		return nil, fmt.Errorf("value is required")
	}

	switch ctx.SecretName {
	case SecretLaunchAgentKey:
		if err := verifyCursorCredentials(ctx.HTTP, v, "", true, false); err != nil {
			return nil, err
		}
	case SecretAdminKey:
		if err := verifyCursorCredentials(ctx.HTTP, "", v, false, true); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}

	return nil, ctx.Secrets.Update(ctx.SecretName, v)
}

func (s *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	requested, ok := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if !ok || len(requested) == 0 {
		return nil, errors.New("no requested capabilities")
	}

	ctx.Capabilities.Request(requested...)

	launchKey, errLaunch := ctx.Secrets.Get(SecretLaunchAgentKey)
	adminKey, errAdmin := ctx.Secrets.Get(SecretAdminKey)
	hasLaunch := errLaunch == nil && strings.TrimSpace(launchKey) != ""
	hasAdmin := errAdmin == nil && strings.TrimSpace(adminKey) != ""

	needLaunch := requestedNeedsLaunchAgentKey(requested)
	needAdmin := requestedNeedsAdminKey(requested)

	if (needLaunch && !hasLaunch) || (needAdmin && !hasAdmin) {
		return s.enterKeysOrDone(ctx.HTTP, ctx.Secrets, ctx.Capabilities, ctx.Capabilities.Requested(), hasLaunch, hasAdmin)
	}

	if err := verifyCursorCredentials(ctx.HTTP, launchKey, adminKey, needLaunch, needAdmin); err != nil {
		return nil, err
	}

	ctx.Capabilities.Enable(requested...)
	return nil, nil
}

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	if len(ctx.Step.Capabilities) == 0 {
		return nil, errors.New("at least one capability is required")
	}

	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	launchKey, errLaunch := ctx.Secrets.Get(SecretLaunchAgentKey)
	adminKey, errAdmin := ctx.Secrets.Get(SecretAdminKey)
	hasLaunch := errLaunch == nil && strings.TrimSpace(launchKey) != ""
	hasAdmin := errAdmin == nil && strings.TrimSpace(adminKey) != ""

	return s.enterKeysOrDone(ctx.HTTP, ctx.Secrets, ctx.Capabilities, ctx.Capabilities.Requested(), hasLaunch, hasAdmin)
}

// enterKeysOrDone returns an enter-keys step when inputs are needed; otherwise verifies stored secrets,
// enables requested capabilities, and returns the done step (skips an empty key form).
func (s *SetupProvider) enterKeysOrDone(
	http core.HTTPContext,
	secrets core.IntegrationSecretStorage,
	capabilities core.CapabilityContext,
	requested []string,
	hasLaunch, hasAdmin bool,
) (*core.SetupStep, error) {
	step := s.enterKeysStep(requested, hasLaunch, hasAdmin)
	if len(step.Inputs) > 0 {
		return step, nil
	}

	launchKey, _ := secrets.Get(SecretLaunchAgentKey)
	adminKey, _ := secrets.Get(SecretAdminKey)
	needLaunch := requestedNeedsLaunchAgentKey(requested)
	needAdmin := requestedNeedsAdminKey(requested)

	if err := verifyCursorCredentials(http, launchKey, adminKey, needLaunch, needAdmin); err != nil {
		return nil, err
	}

	capabilities.Enable(capabilities.Requested()...)
	return cursorSetupDoneStep(), nil
}

func cursorSetupDoneStep() *core.SetupStep {
	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         SetupStepDone,
		Label:        "Setup complete",
		Instructions: "Your Cursor integration is ready. You can use the selected actions in workflows.",
	}
}

func (s *SetupProvider) enterKeysStep(requested []string, hasLaunch, hasAdmin bool) *core.SetupStep {
	needLaunch := requestedNeedsLaunchAgentKey(requested)
	needAdmin := requestedNeedsAdminKey(requested)

	inputs := []configuration.Field{}
	if needLaunch && !hasLaunch {
		inputs = append(inputs, configuration.Field{
			Name:        SecretLaunchAgentKey,
			Label:       "Cloud Agent API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Required for launching Cloud Agents and related actions.",
		})
	}
	if needAdmin && !hasAdmin {
		inputs = append(inputs, configuration.Field{
			Name:        SecretAdminKey,
			Label:       "Admin API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Required for team usage data (Admin API).",
		})
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         SetupStepEnterKeys,
		Label:        "Enter Cursor API keys",
		Inputs:       inputs,
		Instructions: (&Cursor{}).Instructions(),
	}
}

func (s *SetupProvider) onEnterKeysSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	requested := ctx.Capabilities.Requested()
	needLaunch := requestedNeedsLaunchAgentKey(requested)
	needAdmin := requestedNeedsAdminKey(requested)

	launchVal := ""
	if v, ok := m[SecretLaunchAgentKey].(string); ok {
		launchVal = strings.TrimSpace(v)
	}
	adminVal := ""
	if v, ok := m[SecretAdminKey].(string); ok {
		adminVal = strings.TrimSpace(v)
	}

	if needLaunch && launchVal == "" {
		if _, err := ctx.Secrets.Get(SecretLaunchAgentKey); err != nil {
			return nil, errors.New("cloud agent API key is required")
		}
	}
	if needAdmin && adminVal == "" {
		if _, err := ctx.Secrets.Get(SecretAdminKey); err != nil {
			return nil, errors.New("admin API key is required")
		}
	}

	if launchVal != "" {
		if err := persistSecret(ctx, core.IntegrationSecretDefinition{
			Name:        SecretLaunchAgentKey,
			Label:       "Cloud Agent API Key",
			Description: "API key for Cursor Cloud Agents",
			Value:       launchVal,
			Editable:    true,
		}); err != nil {
			return nil, err
		}
	}
	if adminVal != "" {
		if err := persistSecret(ctx, core.IntegrationSecretDefinition{
			Name:        SecretAdminKey,
			Label:       "Admin API Key",
			Description: "API key for Cursor Admin / usage endpoints",
			Value:       adminVal,
			Editable:    true,
		}); err != nil {
			return nil, err
		}
	}

	launchKey, _ := ctx.Secrets.Get(SecretLaunchAgentKey)
	adminKey, _ := ctx.Secrets.Get(SecretAdminKey)

	if err := verifyCursorCredentials(ctx.HTTP, launchKey, adminKey, needLaunch, needAdmin); err != nil {
		return nil, err
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	return cursorSetupDoneStep(), nil
}

func persistSecret(ctx core.SetupStepContext, def core.IntegrationSecretDefinition) error {
	if _, err := ctx.Secrets.Get(def.Name); err != nil {
		return ctx.Secrets.Create(def)
	}
	return ctx.Secrets.Update(def.Name, def.Value)
}
