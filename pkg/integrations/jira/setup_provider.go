package jira

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// Property and secret names used by the Jira OAuth 2.0 (3LO) setup flow.
// Client credentials come from an Atlassian OAuth app the user registers
// themselves in the Developer Console - SuperPlane does not own a shared
// Jira app the way it does for GitHub (whose App Manifest flow creates one
// dynamically), so the client id/secret are entered as the first step.
const (
	PropertyClientID   = "clientId"
	PropertyCloudID    = "cloudId"
	PropertySiteURL    = "siteUrl"
	PropertySiteName   = "siteName"
	PropertyOAuthState = "oauthState"

	SecretOAuthClientSecret = "clientSecret"
	SecretOAuthAccessToken  = "accessToken"
	SecretOAuthRefreshToken = "refreshToken"

	SetupStepEnterAppCredentials = "enterAppCredentials"
	SetupStepAuthorize           = "authorize"
)

type SetupProvider struct{}

// redirectURI is where Atlassian sends the browser back to after the user
// authorizes the app - it must be registered as the callback URL on the
// Atlassian OAuth app itself, and is stable per integration instance.
func redirectURI(baseURL, integrationID string) string {
	return fmt.Sprintf("%s/api/v1/integrations/%s/redirect", baseURL, integrationID)
}

func (s *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	j := &Jira{}
	capabilities := make([]core.Capability, 0, len(j.Actions())+len(j.Triggers()))

	for _, action := range j.Actions() {
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

	for _, trigger := range j.Triggers() {
		capabilities = append(capabilities, core.Capability{
			Type:          core.IntegrationCapabilityTypeTrigger,
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Configuration: trigger.Configuration(),
			ExampleData:   trigger.ExampleData(),
		})
	}

	return []core.CapabilityGroup{
		{Label: "Jira", Capabilities: capabilities},
	}
}

func (s *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	callbackURL := redirectURI(ctx.BaseURL, ctx.IntegrationID.String())

	return core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepEnterAppCredentials,
		Label: "Enter your Atlassian OAuth app credentials",
		Instructions: fmt.Sprintf(`To connect Jira via OAuth 2.0 (3LO):

1. Open the [Atlassian Developer Console](https://developer.atlassian.com/console/myapps/) and create an **OAuth 2.0 (3LO)** app.
2. Add the **Jira API**, with these scopes: `+"`read:jira-work`"+`, `+"`write:jira-work`"+`, `+"`manage:jira-webhook`"+`, `+"`read:jira-user`"+`, `+"`offline_access`"+`.
3. Under **Authorization**, set the callback URL to:

`+"`%s`"+`

4. Copy the app's **Client ID** and **Client Secret** and paste them below.`, callbackURL),
		Inputs: []configuration.Field{
			{
				Name:     PropertyClientID,
				Label:    "Client ID",
				Type:     configuration.FieldTypeString,
				Required: true,
			},
			{
				Name:      SecretOAuthClientSecret,
				Label:     "Client Secret",
				Type:      configuration.FieldTypeString,
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func (s *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step.Name {
	case SetupStepEnterAppCredentials:
		return s.onEnterAppCredentialsSubmit(ctx)

	//
	// This step is not submitted - it's completed by Atlassian redirecting
	// the browser back to HandleRequest's "/redirect" handler.
	//
	case SetupStepAuthorize:
		return nil, nil
	}

	return nil, errors.New("unknown step")
}

func (s *SetupProvider) onEnterAppCredentialsSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	inputs, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	clientID, _ := inputs[PropertyClientID].(string)
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, errors.New("client id is required")
	}

	clientSecret, _ := inputs[SecretOAuthClientSecret].(string)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return nil, errors.New("client secret is required")
	}

	err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:     PropertyClientID,
		Label:    "OAuth Client ID",
		Type:     core.IntegrationPropertyTypeString,
		Value:    clientID,
		Editable: false,
	})
	if err != nil {
		return nil, fmt.Errorf("error saving client id: %w", err)
	}

	err = ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:     SecretOAuthClientSecret,
		Label:    "OAuth Client Secret",
		Value:    clientSecret,
		Editable: false,
	})
	if err != nil {
		return nil, fmt.Errorf("error saving client secret: %w", err)
	}

	state, err := crypto.Base64String(32)
	if err != nil {
		return nil, fmt.Errorf("error generating OAuth state: %w", err)
	}

	err = ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:     PropertyOAuthState,
		Label:    "OAuth State",
		Type:     core.IntegrationPropertyTypeString,
		Value:    state,
		Editable: false,
	})
	if err != nil {
		return nil, fmt.Errorf("error saving OAuth state: %w", err)
	}

	authorizeURL := BuildAuthorizeURL(clientID, redirectURI(ctx.BaseURL, ctx.IntegrationID.String()), state)

	return &core.SetupStep{
		Type:         core.SetupStepTypeRedirectPrompt,
		Name:         SetupStepAuthorize,
		Label:        "Authorize SuperPlane in Atlassian",
		Instructions: "You'll be redirected to Atlassian to authorize SuperPlane's access to your Jira site, then back here.",
		RedirectPrompt: &core.RedirectPrompt{
			URL:    authorizeURL,
			Method: "GET",
		},
	}, nil
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepEnterAppCredentials:
		return ctx.Properties.Delete(PropertyClientID, PropertyOAuthState)
	case SetupStepAuthorize:
		return nil
	}

	return errors.New("unknown step")
}

func (s *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("no property updates are supported for Jira")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("no secret updates are supported for Jira; reconnect the integration to rotate credentials")
}

func (s *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	requested := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if len(requested) == 0 {
		return nil, nil
	}

	//
	// Unlike GitHub, Jira's OAuth scopes are fixed and requested once up
	// front (see oauthScopes), so there's no per-capability permission
	// diffing here - anything requested is immediately enabled.
	//
	ctx.Capabilities.Enable(requested...)
	return nil, nil
}
