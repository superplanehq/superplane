package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/google/go-github/v84/github"
	gh "github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2"
)

type SetupProvider struct{}

const (
	SetupStepSelectOwner      = "selectOwner"
	SetupStepSelectAuthMethod = "selectAuthMethod"
	SetupStepEnterPAT         = "enterPAT"
	SetupStepSetupApp         = "setupApp"

	//
	// An integration can be connected to:
	// - A user account
	// - An organization
	//
	ParameterOwnerType    = "ownerType"
	ParameterOwner        = "owner"
	OwnerTypeUser         = "User Account"
	OwnerTypeOrganization = "Organization"

	//
	// When connecting to the owner with a GitHub App,
	// these are the parameters we get back from GitHub,
	// through the app creation / installation flow.
	//
	ParameterGitHubAppID             = "GitHub App ID"
	ParameterGitHubAppInstallationID = "GitHub App Installation ID"

	//
	// Two authentication methods are supported:
	// - Personal Access Token (PAT)
	// - GitHub App
	//
	ParameterAuthMethod = "Authentication Method"
	AuthMethodPAT       = "Personal Access Token"
	AuthMethodGitHubApp = "GitHub App"

	//
	// Secrets for the integration:
	// - Personal Access Token (PAT)
	// - GitHub App private key (PEM)
	//
	SecretPAT          = "Personal Access Token"
	SecretGitHubAppPEM = "GitHub App Private Key (PEM)"
)

func (g *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			Label:        "Actions",
			Capabilities: append(g.readActionCapabilities(), g.writeActionCapabilities()...),
		},
		{
			Label:        "Commit Statuses",
			Capabilities: g.writeCommitStatusCapabilities(),
		},
		{
			Label:        "Contents",
			Capabilities: append(g.readContentsCapabilities(), g.writeContentsCapabilities()...),
		},
		{
			Label:        "Issues",
			Capabilities: append(g.readIssueCapabilities(), g.writeIssueCapabilities()...),
		},
		{
			Label:        "Pull Requests",
			Capabilities: append(g.readPullRequestCapabilities(), g.writePullRequestCapabilities()...),
		},
	}
}

func (g *SetupProvider) isIssueCapability(capabilityName string) (bool, bool) {
	if slices.ContainsFunc(g.writeIssueCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, true
	}

	if slices.ContainsFunc(g.readIssueCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, false
	}

	return false, false
}

func (g *SetupProvider) readIssueCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&GetIssue{},
		},
		[]core.Trigger{
			&OnIssue{},
			&OnIssueComment{},
		},
	)
}

func (g *SetupProvider) writeIssueCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&CreateIssue{},
			&CreateIssueComment{},
			&UpdateIssue{},
			&RemoveIssueLabel{},
			&RemoveIssueAssignee{},
			&AddIssueLabel{},
			&AddIssueAssignee{},
		},
		[]core.Trigger{},
	)
}

func (g *SetupProvider) isPullRequestCapability(capabilityName string) (bool, bool) {
	if slices.ContainsFunc(g.writePullRequestCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, true
	}

	if slices.ContainsFunc(g.readPullRequestCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, false
	}

	return false, false
}

func (g *SetupProvider) readPullRequestCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{},
		[]core.Trigger{
			&OnPullRequest{},
			&OnPRComment{},
			&OnPRReviewComment{},
		},
	)
}

func (g *SetupProvider) writePullRequestCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&CreateReview{},
			&AddReaction{},
		},
		[]core.Trigger{},
	)
}

func (g *SetupProvider) isActionCapability(capabilityName string) (bool, bool) {
	if slices.ContainsFunc(g.writeActionCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, true
	}

	if slices.ContainsFunc(g.readActionCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, false
	}

	return false, false
}

func (g *SetupProvider) readActionCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&GetWorkflowUsage{},
		},
		[]core.Trigger{
			&OnWorkflowRun{},
		},
	)
}

func (g *SetupProvider) writeActionCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&RunWorkflow{},
		},
		[]core.Trigger{},
	)
}

func (g *SetupProvider) isCommitStatusCapability(capabilityName string) (bool, bool) {
	if slices.ContainsFunc(g.writeCommitStatusCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, true
	}

	return false, false
}

func (g *SetupProvider) writeCommitStatusCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&PublishCommitStatus{},
		},
		[]core.Trigger{},
	)
}

func (g *SetupProvider) isContentsCapability(capabilityName string) (bool, bool) {
	if slices.ContainsFunc(g.writeContentsCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, true
	}

	if slices.ContainsFunc(g.readContentsCapabilities(), func(capability core.Capability) bool {
		return capability.Name == capabilityName
	}) {
		return true, false
	}

	return false, false
}

func (g *SetupProvider) readContentsCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&GetRelease{},
			&GetRepositoryPermission{},
		},
		[]core.Trigger{
			&OnBranchCreated{},
			&OnPush{},
			&OnRelease{},
			&OnTagCreated{},
		},
	)
}

func (g *SetupProvider) writeContentsCapabilities() []core.Capability {
	return g.genCapabilities(
		[]core.Action{
			&CreateRelease{},
			&UpdateRelease{},
			&DeleteRelease{},
		},
		[]core.Trigger{},
	)
}

func (g *SetupProvider) Capabilities() []core.Capability {
	capabilities := []core.Capability{}
	for _, group := range g.CapabilityGroups() {
		capabilities = append(capabilities, group.Capabilities...)
	}
	return capabilities
}

func (g *SetupProvider) genCapabilities(actions []core.Action, triggers []core.Trigger) []core.Capability {
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

func (g *SetupProvider) OnParameterUpdate(ctx core.ParameterUpdateContext) (*core.SetupStep, error) {
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
				Name:     ParameterOwnerType,
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
				Name:        ParameterOwner,
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
	return ctx.Parameters.Delete(ParameterOwnerType, ParameterOwner)
}

func (g *SetupProvider) onSelectAuthMethodRevert(ctx core.SetupStepContext) error {
	return ctx.Parameters.Delete(ParameterAuthMethod)
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
	case SetupStepSetupApp:
		return nil, fmt.Errorf("not implemented")
	}

	return nil, errors.New("unknown step")
}

func (g *SetupProvider) onSelectOwnerSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	owner, ok := m[ParameterOwner].(string)
	if !ok {
		return nil, errors.New("invalid owner")
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, errors.New("owner is required")
	}

	err := ctx.Parameters.Create(core.IntegrationParameterDefinition{
		Name:     ParameterOwner,
		Label:    "Owner",
		Type:     configuration.FieldTypeString,
		Value:    owner,
		Editable: false,
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
				Name:     ParameterAuthMethod,
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

	authMethod, ok := m[ParameterAuthMethod].(string)
	if !ok {
		return nil, errors.New("invalid authentication method")
	}

	if authMethod != AuthMethodPAT && authMethod != AuthMethodGitHubApp {
		return nil, errors.New("invalid authentication method")
	}

	err := ctx.Parameters.Create(core.IntegrationParameterDefinition{
		Name:     ParameterAuthMethod,
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
		return nil, fmt.Errorf("TODO: implement GitHub app authentication")

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
	owner, err := ctx.Parameters.GetString(ParameterOwner)
	if err != nil {
		return "", err
	}

	permissions, err := g.getPermissions(ctx)
	if err != nil {
		return "", err
	}

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

func (g *SetupProvider) getPermissions(ctx core.SetupStepContext) (map[string]string, error) {
	requestedCapabilities := ctx.Capabilities.Requested()
	if len(requestedCapabilities) == 0 {
		return nil, fmt.Errorf("no capabilities requested")
	}

	//
	// TODO: this looks awful, need a better way to build this kind of logic.
	//
	permissions := NewPermissionSet()
	for _, capability := range requestedCapabilities {
		isRead, isWrite := g.isIssueCapability(capability)
		if isRead {
			permissions.Add("Issues", false)
		}
		if isWrite {
			permissions.Add("Issues", true)
		}
		isRead, isWrite = g.isPullRequestCapability(capability)
		if isRead {
			permissions.Add("Pull Requests", false)
		}
		if isWrite {
			permissions.Add("Pull Requests", true)
		}
		isRead, isWrite = g.isActionCapability(capability)
		if isRead {
			permissions.Add("Actions", false)
		}
		if isWrite {
			permissions.Add("Actions", true)
		}

		isRead, isWrite = g.isCommitStatusCapability(capability)
		if isRead {
			permissions.Add("Commit Statuses", false)
		}
		if isWrite {
			permissions.Add("Commit Statuses", true)
		}

		isRead, isWrite = g.isContentsCapability(capability)
		if isRead {
			permissions.Add("Contents", false)
		}
		if isWrite {
			permissions.Add("Contents", true)
		}
	}

	return permissions.Permissions(), nil
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

	err := ctx.Secrets.Create(SecretPAT, core.IntegrationSecretDefinition{
		Value:    []byte(token),
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

	// TODO: enable capabilities

	return finishPATSetup(ctx.Parameters, repos)
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

func finishPATSetup(parameters core.IntegrationParameterStorage, repos []*github.Repository) (*core.SetupStep, error) {
	owner, err := parameters.GetString(ParameterOwner)
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
