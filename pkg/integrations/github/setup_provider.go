package github

import (
	"bytes"
	"context"
	_ "embed"
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
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"golang.org/x/oauth2"
)

type SetupProvider struct{}

//go:embed templates/pat-instructions.tpl
var patInstructionsTemplate []byte

//go:embed templates/pat-update-instructions.tpl
var patUpdateInstructionsTemplate []byte

//go:embed templates/setup-complete.tpl
var setupCompletedTemplate []byte

const (
	SetupStepSelectOwner       = "selectOwner"
	SetupStepSelectAuthMethod  = "selectAuthMethod"
	SetupStepEnterPAT          = "enterPAT"
	SetupStepUpdatePermissions = "updatePermissions"
	SetupStepSetupApp          = "setupApp"
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

func (g *SetupProvider) instructionsForTokenUpdate(properties core.IntegrationPropertyStorageReader, newCapabilities []string) (string, error) {
	owner, err := properties.GetString(common.PropertyOwner)
	if err != nil {
		return "", err
	}

	permissions := NewCapabilityMapper().PermissionsForPAT(newCapabilities)
	tmpl, err := template.New("patUpdateInstructions").Parse(string(patUpdateInstructionsTemplate))
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
				Name:     common.PropertyOwnerType,
				Label:    "Owner Type",
				Type:     configuration.FieldTypeSelect,
				Required: true,
				Default:  common.OwnerTypeUser,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "User Account", Value: common.OwnerTypeUser},
							{Label: "Organization", Value: common.OwnerTypeOrganization},
						},
					},
				},
			},
			{
				Name:        common.PropertyOwner,
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
	return ctx.Properties.Delete(common.PropertyOwnerType, common.PropertyOwner)
}

func (g *SetupProvider) onSelectAuthMethodRevert(ctx core.SetupStepContext) error {
	return ctx.Properties.Delete(common.PropertyAuthMethod)
}

func (g *SetupProvider) onEnterPATRevert(ctx core.SetupStepContext) error {
	return ctx.Secrets.Delete(common.SecretPAT)
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

	ownerType, ok := m[common.PropertyOwnerType].(string)
	if !ok {
		return nil, errors.New("invalid owner type")
	}

	if ownerType != common.OwnerTypeUser && ownerType != common.OwnerTypeOrganization {
		return nil, errors.New("invalid owner type")
	}

	owner, ok := m[common.PropertyOwner].(string)
	if !ok {
		return nil, errors.New("invalid owner")
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, errors.New("owner is required")
	}

	err := ctx.Properties.CreateMany([]core.IntegrationPropertyDefinition{
		{
			Name:     common.PropertyOwnerType,
			Label:    "Owner Type",
			Type:     configuration.FieldTypeString,
			Value:    ownerType,
			Editable: false,
		},
		{
			Name:     common.PropertyOwner,
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
				Name:     common.PropertyAuthMethod,
				Label:    "Authentication Method",
				Type:     configuration.FieldTypeSelect,
				Required: true,
				Default:  common.AuthMethodPAT,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "Personal Access Token", Value: common.AuthMethodPAT},
							{Label: "GitHub App", Value: common.AuthMethodGitHubApp},
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

	authMethod, ok := m[common.PropertyAuthMethod].(string)
	if !ok {
		return nil, errors.New("invalid authentication method")
	}

	if authMethod != common.AuthMethodPAT && authMethod != common.AuthMethodGitHubApp {
		return nil, errors.New("invalid authentication method")
	}

	err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:     common.PropertyAuthMethod,
		Label:    "Authentication Method",
		Type:     configuration.FieldTypeString,
		Value:    authMethod,
		Editable: false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	switch authMethod {
	case common.AuthMethodPAT:
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
					Name:      common.SecretPAT,
					Label:     "Personal Access Token",
					Type:      configuration.FieldTypeString,
					Required:  true,
					Sensitive: true,
				},
			},
			Instructions: instructions,
		}, nil

	case common.AuthMethodGitHubApp:
		state, err := crypto.Base64String(32)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate GitHub App state: %v", err)
		}

		err = ctx.Properties.Create(core.IntegrationPropertyDefinition{
			Name:     common.PropertyAppState,
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

func (g *SetupProvider) generateInstructionsForPAT(ctx core.SetupStepContext) (string, error) {
	owner, err := ctx.Properties.GetString(common.PropertyOwner)
	if err != nil {
		return "", err
	}

	requestedCapabilities := ctx.Capabilities.Requested()
	if len(requestedCapabilities) == 0 {
		return "", fmt.Errorf("no capabilities requested")
	}

	permissions := NewCapabilityMapper().PermissionsForPAT(requestedCapabilities)
	tmpl, err := template.New("patInstructions").Parse(string(patInstructionsTemplate))
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
	ownerType, err := ctx.Properties.GetString(common.PropertyOwnerType)
	if err != nil {
		return nil, err
	}

	owner, err := ctx.Properties.GetString(common.PropertyOwner)
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
	if ownerType == common.OwnerTypeOrganization {
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

	token, ok := m[common.SecretPAT].(string)
	if !ok {
		return nil, errors.New("invalid personal access token")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("personal access token is required")
	}

	err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:     common.SecretPAT,
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

func finishPATSetup(properties core.IntegrationPropertyStorageReader, repos []*github.Repository) (*core.SetupStep, error) {
	owner, err := properties.GetString(common.PropertyOwner)
	if err != nil {
		return nil, fmt.Errorf("error getting connection URL: %v", err)
	}

	tmpl, err := template.New("setupCompleted").Parse(string(setupCompletedTemplate))
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
