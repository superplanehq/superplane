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
	SetupStepEnterLaunchKey      = "enterLaunchKey"
	SetupStepEnterAdminKey       = "enterAdminKey"
	SetupStepDone                = "done"
)

const (
	SecretLaunchAgentKey = "launchAgentKey"
	SecretAdminKey       = "adminKey"
)

type SetupProvider struct{}

func (s *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			// Things that create/manage Cursor Cloud Agents.
			// Future-friendly examples: list agents, cancel agent, get agent status, get conversation, etc.
			Label: "Agents",
			Capabilities: core.BuildCapabilities(
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
			Capabilities: core.BuildCapabilities(
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
	case SetupStepEnterLaunchKey:
		return s.onEnterLaunchKeySubmit(ctx.Step.Inputs, ctx)
	case SetupStepEnterAdminKey:
		return s.onEnterAdminKeySubmit(ctx.Step.Inputs, ctx)
	default:
		return nil, errors.New("unknown step")
	}
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case SetupStepEnterLaunchKey, SetupStepEnterAdminKey:
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
		return s.nextStepOrDone(ctx.HTTP, ctx.Secrets, ctx.Capabilities, ctx.Capabilities.Requested(), hasLaunch, hasAdmin)
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
	ctx.Capabilities.Available(core.CapabilityNamesNotRequested(s.CapabilityGroups(), ctx.Step.Capabilities)...)

	launchKey, errLaunch := ctx.Secrets.Get(SecretLaunchAgentKey)
	adminKey, errAdmin := ctx.Secrets.Get(SecretAdminKey)
	hasLaunch := errLaunch == nil && strings.TrimSpace(launchKey) != ""
	hasAdmin := errAdmin == nil && strings.TrimSpace(adminKey) != ""

	return s.nextStepOrDone(ctx.HTTP, ctx.Secrets, ctx.Capabilities, ctx.Capabilities.Requested(), hasLaunch, hasAdmin)
}

// nextStepOrDone returns the next missing-key step (launch then admin), otherwise verifies stored secrets,
// enables requested capabilities, and returns the done step.
func (s *SetupProvider) nextStepOrDone(
	http core.HTTPContext,
	secrets core.IntegrationSecretStorage,
	capabilities core.CapabilityContext,
	requested []string,
	hasLaunch, hasAdmin bool,
) (*core.SetupStep, error) {
	needLaunch := requestedNeedsLaunchAgentKey(requested)
	needAdmin := requestedNeedsAdminKey(requested)

	if needLaunch && !hasLaunch {
		return s.enterLaunchKeyStep(), nil
	}
	if needAdmin && !hasAdmin {
		return s.enterAdminKeyStep(), nil
	}

	launchKey, _ := secrets.Get(SecretLaunchAgentKey)
	adminKey, _ := secrets.Get(SecretAdminKey)

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

func (s *SetupProvider) enterLaunchKeyStep() *core.SetupStep {
	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterLaunchKey,
		Label: "Enter Cloud Agent API key",
		Inputs: []configuration.Field{
			{
				Name:        SecretLaunchAgentKey,
				Label:       "Cloud Agent API Key",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Required for launching Cloud Agents and related actions.",
			},
		},
		Instructions: "Create or copy a Cloud Agent API key from the Cursor Dashboard.",
	}
}

func (s *SetupProvider) enterAdminKeyStep() *core.SetupStep {
	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterAdminKey,
		Label: "Enter Admin API key",
		Inputs: []configuration.Field{
			{
				Name:        SecretAdminKey,
				Label:       "Admin API Key",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Required for team usage data (Admin API).",
			},
		},
		Instructions: "Create or copy an Admin API key from the Cursor Dashboard (requires appropriate org/team permissions).",
	}
}

func (s *SetupProvider) onEnterLaunchKeySubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
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

	if needLaunch && launchVal == "" {
		return nil, errors.New("cloud agent API key is required")
	}

	if needLaunch {
		if err := verifyCursorCredentials(ctx.HTTP, launchVal, "", true, false); err != nil {
			return nil, err
		}
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

	// If only the launch key is required, we already verified it above.
	if needLaunch && !needAdmin {
		ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
		return cursorSetupDoneStep(), nil
	}

	hasAdmin := false
	if v, err := ctx.Secrets.Get(SecretAdminKey); err == nil && strings.TrimSpace(v) != "" {
		hasAdmin = true
	}
	if needAdmin && !hasAdmin {
		return s.enterAdminKeyStep(), nil
	}

	if needAdmin {
		adminKey, _ := ctx.Secrets.Get(SecretAdminKey)
		// Launch key was already verified above when needLaunch; verify only the admin key here.
		if err := verifyCursorCredentials(ctx.HTTP, "", adminKey, false, true); err != nil {
			return nil, err
		}
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	return cursorSetupDoneStep(), nil
}

func (s *SetupProvider) onEnterAdminKeySubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	requested := ctx.Capabilities.Requested()
	needAdmin := requestedNeedsAdminKey(requested)

	adminVal := ""
	if v, ok := m[SecretAdminKey].(string); ok {
		adminVal = strings.TrimSpace(v)
	}
	if needAdmin && adminVal == "" {
		return nil, errors.New("admin API key is required")
	}

	if err := verifyCursorCredentials(ctx.HTTP, "", adminVal, false, true); err != nil {
		return nil, err
	}

	if err := persistSecret(ctx, core.IntegrationSecretDefinition{
		Name:        SecretAdminKey,
		Label:       "Admin API Key",
		Description: "API key for Cursor Admin / usage endpoints",
		Value:       adminVal,
		Editable:    true,
	}); err != nil {
		return nil, err
	}

	if requestedNeedsLaunchAgentKey(requested) {
		launchKey, _ := ctx.Secrets.Get(SecretLaunchAgentKey)
		if err := verifyCursorCredentials(ctx.HTTP, launchKey, "", true, false); err != nil {
			return nil, err
		}
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
