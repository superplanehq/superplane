package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/go-github/v84/github"
	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"golang.org/x/oauth2"
)

type SetupProvider struct{}

const (
	SetupStepSelectOwner       = "selectOwner"
	SetupStepSelectAuthMethod  = "selectAuthMethod"
	SetupStepEnterPAT          = "enterPAT"
	SetupStepUpdatePermissions = "updatePermissions"
	SetupStepSetupApp          = "setupApp"

	//
	// An integration can be connected to:
	// - A user account
	// - An organization
	//
	PropertyOwnerType     = "ownerType"
	PropertyOwner         = "owner"
	OwnerTypeUser         = "User Account"
	OwnerTypeOrganization = "Organization"

	//
	// When connecting to the owner with a GitHub App,
	// these are the properties we get back from GitHub,
	// through the app creation / installation flow.
	//
	PropertyAppID             = "GitHub App ID"
	PropertyAppSlug           = "GitHub App Slug"
	PropertyAppClientID       = "GitHub App Client ID"
	PropertyAppInstallationID = "GitHub App Installation ID"
	PropertyAppState          = "GitHub App State"

	//
	// Two authentication methods are supported:
	// - Personal Access Token (PAT)
	// - GitHub App
	//
	PropertyAuthMethod  = "Authentication Method"
	AuthMethodPAT       = "Personal Access Token"
	AuthMethodGitHubApp = "GitHub App"

	//
	// Secrets for the integration:
	// - Personal Access Token (PAT)
	// - GitHub App private key (PEM)
	//
	SecretPAT              = "Personal Access Token"
	SecretAppClientSecret  = "GitHub App Client Secret"
	SecretAppWebhookSecret = "GitHub App Webhook Secret"
	SecretAppPEM           = "GitHub App Private Key (PEM)"
)

func (g *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	changes := ctx.Changes
	if len(changes) == 0 {
		return nil, nil
	}

	requested := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if len(requested) == 0 {
		return nil, errors.New("no requested capabilities")
	}

	instructions, err := g.instructionsForTokenUpdate(ctx.Properties, requested)
	if err != nil {
		return nil, fmt.Errorf("error generating instructions: %v", err)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         SetupStepUpdatePermissions,
		Label:        "Update the permissions on your personal access token",
		Inputs:       []configuration.Field{},
		Instructions: instructions,
	}, nil
}

const patUpdateInstructionsTemplate = `
- Go to https://github.com/settings/personal-access-tokens
- Find the token you want to update
- Edit the permissions to include the following:

| Resource | Permission Level |
|---------|------------|
{{- range $key, $value := .Permissions }}
| {{ $key }} | {{ $value }} |
{{- end }}
`

func (g *SetupProvider) instructionsForTokenUpdate(properties core.IntegrationPropertyStorageReader, newCapabilities []string) (string, error) {
	owner, err := properties.GetString(PropertyOwner)
	if err != nil {
		return "", err
	}

	permissions := NewCapabilityMapper().PermissionsForPAT(newCapabilities)
	tmpl, err := template.New("patUpdateInstructions").Parse(patUpdateInstructionsTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"Owner":       owner,
		"Permissions": permissions,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}

func (g *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	mapper := NewCapabilityMapper()
	groups := []core.CapabilityGroup{}
	for name, group := range mapper.Groups {
		capabilities := []core.Capability{}
		for _, c := range group {
			if c.Action != nil {
				capabilities = append(capabilities, core.Capability{
					Type:           core.IntegrationCapabilityTypeAction,
					Name:           c.Action.Name(),
					Label:          c.Action.Label(),
					Description:    c.Action.Description(),
					Configuration:  c.Action.Configuration(),
					OutputChannels: c.Action.OutputChannels(nil),
				})
			}

			if c.Trigger != nil {
				capabilities = append(capabilities, core.Capability{
					Type:          core.IntegrationCapabilityTypeTrigger,
					Name:          c.Trigger.Name(),
					Label:         c.Trigger.Label(),
					Description:   c.Trigger.Description(),
					Configuration: c.Trigger.Configuration(),
				})
			}
		}

		groups = append(groups, core.CapabilityGroup{
			Label:        name,
			Capabilities: capabilities,
		})
	}

	return groups
}

func (g *SetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("TODO")
}

func (g *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("TODO")
}

func (g *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	return core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectOwner,
		Label: "Select the user account / organization",
		Inputs: []configuration.Field{
			{
				Name:     PropertyOwnerType,
				Label:    "Owner Type",
				Type:     configuration.FieldTypeSelect,
				Required: true,
				Default:  OwnerTypeUser,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "User Account", Value: OwnerTypeUser},
							{Label: "Organization", Value: OwnerTypeOrganization},
						},
					},
				},
			},
			{
				Name:        PropertyOwner,
				Label:       "User account / organization name",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Placeholder: "e.g. superplanehq",
			},
		},
	}
}

func (g *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step {
	case SetupStepSelectOwner:
		return g.onSelectOwnerRevert(ctx)
	case SetupStepSelectAuthMethod:
		return g.onSelectAuthMethodRevert(ctx)
	case SetupStepEnterPAT:
		return g.onEnterPATRevert(ctx)
	case SetupStepSetupApp:
		return fmt.Errorf("not implemented")
	}

	return errors.New("unknown step")
}

func (g *SetupProvider) onSelectOwnerRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete(PropertyOwnerType, PropertyOwner)
}

func (g *SetupProvider) onSelectAuthMethodRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete(PropertyAuthMethod)
}

func (g *SetupProvider) onEnterPATRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete(SecretPAT)
}

func (g *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step {
	case SetupStepSelectOwner:
		return g.onSelectOwnerSubmit(ctx.Inputs, ctx)

	case SetupStepSelectAuthMethod:
		return g.onSelectAuthMethodSubmit(ctx.Inputs, ctx)

	case SetupStepEnterPAT:
		return g.onEnterPATSubmit(ctx.Inputs, ctx)

	case SetupStepUpdatePermissions:
		err := ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
		if err != nil {
			return nil, fmt.Errorf("error enabling capabilities: %v", err)
		}

		return nil, nil

	//
	// This step is not submitted, since it's a redirect step.
	// The GitHub App creation flow will clear the setup state, if successful.
	//
	case SetupStepSetupApp:
		return nil, nil
	}

	return nil, errors.New("unknown step")
}

func (g *SetupProvider) onSelectOwnerSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	ownerType, ok := m[PropertyOwnerType].(string)
	if !ok {
		return nil, errors.New("invalid owner type")
	}

	if ownerType != OwnerTypeUser && ownerType != OwnerTypeOrganization {
		return nil, errors.New("invalid owner type")
	}

	owner, ok := m[PropertyOwner].(string)
	if !ok {
		return nil, errors.New("invalid owner")
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, errors.New("owner is required")
	}

	err := ctx.Properties.CreateMany([]core.IntegrationPropertyDefinition{
		{
			Name:     PropertyOwnerType,
			Label:    "Owner Type",
			Type:     configuration.FieldTypeString,
			Value:    ownerType,
			Editable: false,
		},
		{
			Name:     PropertyOwner,
			Label:    "Owner",
			Type:     configuration.FieldTypeString,
			Value:    owner,
			Editable: false,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectAuthMethod,
		Label: "Choose authentication method",
		Inputs: []configuration.Field{
			{
				Name:     PropertyAuthMethod,
				Label:    "Authentication Method",
				Type:     configuration.FieldTypeSelect,
				Required: true,
				Default:  AuthMethodPAT,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "Personal Access Token", Value: AuthMethodPAT},
							{Label: "GitHub App", Value: AuthMethodGitHubApp},
						},
					},
				},
			},
		},
	}, nil
}

func (g *SetupProvider) onSelectAuthMethodSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	authMethod, ok := m[PropertyAuthMethod].(string)
	if !ok {
		return nil, errors.New("invalid authentication method")
	}

	if authMethod != AuthMethodPAT && authMethod != AuthMethodGitHubApp {
		return nil, errors.New("invalid authentication method")
	}

	err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:     PropertyAuthMethod,
		Label:    "Authentication Method",
		Type:     configuration.FieldTypeString,
		Value:    authMethod,
		Editable: false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	switch authMethod {
	case AuthMethodPAT:
		instructions, err := g.generateInstructionsForPAT(ctx)
		if err != nil {
			return nil, fmt.Errorf("error generating instructions: %v", err)
		}

		return &core.SetupStep{
			Type:  core.SetupStepTypeInputs,
			Name:  SetupStepEnterPAT,
			Label: "Enter GitHub Personal Access Token",
			Inputs: []configuration.Field{
				{
					Name:      SecretPAT,
					Label:     "Personal Access Token",
					Type:      configuration.FieldTypeString,
					Required:  true,
					Sensitive: true,
				},
			},
			Instructions: instructions,
		}, nil

	case AuthMethodGitHubApp:
		state, err := crypto.Base64String(32)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate GitHub App state: %v", err)
		}

		err = ctx.Properties.Create(core.IntegrationPropertyDefinition{
			Name:     PropertyAppState,
			Label:    "GitHub App State",
			Type:     configuration.FieldTypeString,
			Value:    state,
			Editable: false,
		})

		if err != nil {
			return nil, fmt.Errorf("error creating property: %v", err)
		}

		return g.generateNextStepForApp(ctx, state)

	default:
		return nil, fmt.Errorf("not implemented")
	}
}

const patInstructionsTemplate = `
- Go to https://github.com/settings/personal-access-tokens/new
- Generate a new fine-grained personal access token
- **Token name**: ` + "`SuperPlane`" + `
- **Resource owner**: ` + "`{{ .Owner }}`" + `
- **Expiration**: based on your security policy
- Under **Repository access**, choose the repositories SuperPlane should access
- Based on the capabilities you selected, these are the permissions you need to grant to the token:

| Resource | Permission Level |
|---------|------------|
{{- range $key, $value := .Permissions }}
| {{ $key }} | {{ $value }} |
{{- end }}
`

// Helper struct for generating a set of permissions
// based on the capabilities requested.
type PermissionSet struct {

	//
	// Current permissions requested.
	// Map of <resource>:<writable>
	// e.g. "issues:true", "pulls:false", "contents:true"
	// No key in here means the resource was not requested.
	//
	permissions map[string]bool
}

func NewPermissionSet() *PermissionSet {
	return &PermissionSet{
		permissions: map[string]bool{
			"Webhooks": true,
		},
	}
}

func (p *PermissionSet) Add(resource string, writable bool) {
	p.permissions[resource] = writable
}

func (p *PermissionSet) Permissions() map[string]string {
	out := map[string]string{}
	for resource, writable := range p.permissions {
		if writable {
			out[resource] = "Read & Write"
		} else {
			out[resource] = "Read"
		}
	}

	return out
}

func (g *SetupProvider) generateInstructionsForPAT(ctx core.SetupStepContext) (string, error) {
	owner, err := ctx.Properties.GetString(PropertyOwner)
	if err != nil {
		return "", err
	}

	requestedCapabilities := ctx.Capabilities.Requested()
	if len(requestedCapabilities) == 0 {
		return "", fmt.Errorf("no capabilities requested")
	}

	permissions := NewCapabilityMapper().PermissionsForPAT(requestedCapabilities)
	tmpl, err := template.New("patInstructions").Parse(patInstructionsTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"Owner":       owner,
		"Permissions": permissions,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}

func (g *SetupProvider) generateNextStepForApp(ctx core.SetupStepContext, state string) (*core.SetupStep, error) {
	ownerType, err := ctx.Properties.GetString(PropertyOwnerType)
	if err != nil {
		return nil, err
	}

	owner, err := ctx.Properties.GetString(PropertyOwner)
	if err != nil {
		return nil, err
	}

	return &core.SetupStep{
		Type:  core.SetupStepTypeRedirectPrompt,
		Name:  SetupStepSetupApp,
		Label: "Setup GitHub App",
		RedirectPrompt: &core.RedirectPrompt{
			Method: "POST",
			URL:    g.createAppURL(ownerType, owner),
			FormData: map[string]string{
				"manifest": g.appManifest(ctx),
				"state":    state,
			},
		},
		Instructions: "",
	}, nil
}

func (g *SetupProvider) createAppURL(ownerType string, owner string) string {
	if ownerType == OwnerTypeOrganization {
		return fmt.Sprintf("https://github.com/organizations/%s/settings/apps/new", owner)
	}

	return "https://github.com/settings/apps/new"
}

func (g *SetupProvider) appManifest(ctx core.SetupStepContext) string {
	permissions := NewCapabilityMapper().PermissionsForApp(ctx.Capabilities.Requested())
	manifest := map[string]any{
		"name":                `SuperPlane GH integration`,
		"public":              false,
		"url":                 "https://superplane.com",
		"default_events":      defaultGitHubAppEvents,
		"default_permissions": permissions,
		"setup_url":           fmt.Sprintf(`%s/api/v1/integrations/%s/setup`, ctx.BaseURL, ctx.IntegrationID),
		"redirect_url":        fmt.Sprintf(`%s/api/v1/integrations/%s/redirect`, ctx.BaseURL, ctx.IntegrationID),
		"hook_attributes": map[string]any{
			"url": fmt.Sprintf(`%s/api/v1/integrations/%s/webhook`, ctx.WebhooksBaseURL, ctx.IntegrationID),
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return ""
	}

	return string(data)
}

func (g *SetupProvider) onEnterPATSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	token, ok := m[SecretPAT].(string)
	if !ok {
		return nil, errors.New("invalid personal access token")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("personal access token is required")
	}

	err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:     SecretPAT,
		Value:    token,
		Label:    "Personal Access Token",
		Editable: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	repos, err := validatePATConnection(token)
	if err != nil {
		return nil, err
	}

	err = ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
	if err != nil {
		return nil, fmt.Errorf("error enabling capabilities: %v", err)
	}

	return finishPATSetup(ctx.Properties, repos)
}

func validatePATConnection(token string) ([]*github.Repository, error) {
	client := gh.NewClient(
		oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		),
	)

	repos, _, err := client.Repositories.ListByAuthenticatedUser(
		context.Background(),
		&gh.RepositoryListByAuthenticatedUserOptions{
			Affiliation: "owner",
			Sort:        "updated",
			ListOptions: gh.ListOptions{PerPage: 50},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %v", err)
	}

	return repos, nil
}

const setupCompletedTemplate = `
{{- $connectionURL := .ConnectionURL }}
You are now connected to {{ $connectionURL }}

---

You can now start using the following repositories:

| Name | URL |
|------|-----|
{{- range .Repos }}
| ` + "`{{ .FullName }}`" + ` | {{ .HTMLURL }}
{{- end }}
`

func finishPATSetup(properties core.IntegrationPropertyStorageReader, repos []*github.Repository) (*core.SetupStep, error) {
	owner, err := properties.GetString(PropertyOwner)
	if err != nil {
		return nil, fmt.Errorf("error getting connection URL: %v", err)
	}

	tmpl, err := template.New("setupCompleted").Parse(setupCompletedTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	data := map[string]any{
		"ConnectionURL": fmt.Sprintf("https://github.com/%s", owner),
		"Repos":         repos,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         "done",
		Label:        "GitHub connection completed successfully",
		Instructions: buf.String(),
	}, nil
}
