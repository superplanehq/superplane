package github

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	gh "github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2"
)

type SetupProvider struct{}

const (
	SetupStepSelectOwner      = "selectOwner"
	SetupStepSelectResources  = "selectResources"
	SetupStepSelectAuthMethod = "selectAuthMethod"
	SetupStepEnterPAT         = "enterPAT"
	SetupStepSetupApp         = "setupApp"

	ParameterOwner                   = "owner"
	ParameterResources               = "resourceTypes"
	ParameterAuthMethod              = "authenticationMethod"
	ParameterGitHubAppID             = "githubAppID"
	ParameterGitHubAppInstallationID = "githubAppInstallationID"

	SecretPAT          = "pat"
	SecretGitHubAppPEM = "githubAppPEM"

	AuthMethodPAT       = "pat"
	AuthMethodGitHubApp = "github_app"

	ResourceIssues                = "issues"
	ResourcePullRequests          = "pull_requests"
	ResourceWorkflows             = "workflows"
	ResourceReleases              = "releases"
	ResourceCode                  = "code"
	ResourceRepositoryPermissions = "repository_permissions"
)

var AllResourceTypes = []string{
	ResourceIssues,
	ResourcePullRequests,
	ResourceWorkflows,
	ResourceReleases,
	ResourceCode,
	ResourceRepositoryPermissions,
}

var ResourceTypeOptions = []configuration.FieldOption{
	{Label: "Issues", Value: ResourceIssues},
	{Label: "Pull Requests", Value: ResourcePullRequests},
	{Label: "Workflows (GitHub Actions)", Value: ResourceWorkflows},
	{Label: "Releases", Value: ResourceReleases},
	{Label: "Code (Pushes, Branches, Tags, Commit Status)", Value: ResourceCode},
	{Label: "Repository Access / Permissions", Value: ResourceRepositoryPermissions},
}

const ownerInstructions = `Enter the GitHub login for the account that owns the repositories you want to use in SuperPlane.

Examples:
~~~text
superplanehq
octocat
~~~

You can confirm the owner login in the URL:
~~~text
https://github.com/<owner>
~~~`

const resourcesInstructions = `Select the GitHub resource groups SuperPlane should enable.

These selections define which GitHub components and triggers are available in workflows:

- **Issues**: issue actions and issue-related triggers
- **Pull Requests**: PR reviews, comments, reactions, and PR triggers
- **Workflows (GitHub Actions)**: run and monitor Actions workflows
- **Releases**: create, read, update, and delete releases
- **Code**: push/branch/tag triggers and commit statuses
- **Repository Access / Permissions**: repository permission checks`

const authMethodInstructions = `Choose how SuperPlane authenticates with GitHub.

- **Personal Access Token**: fastest setup for PAT-based access (current flow)
- **Private GitHub App**: recommended for advanced organization setups (not part of this flow yet)`

const patInstructionsTemplate = `Use a **Fine-grained personal access token** and paste it below.

## 1. Create a token

1. Go to [GitHub fine-grained token settings](https://github.com/settings/personal-access-tokens/new)
2. Click **Generate new token**
3. Set **Token name** to something like ` + "`SuperPlane`" + `
4. Set **Resource owner** to **%s**
5. Set **Expiration** based on your security policy
6. Under **Repository access**, choose the repositories SuperPlane should access

## 2. Set permissions

Selected resource groups:

~~~text
%s
~~~

Set at least these repository permissions:

~~~text
%s
~~~

## 3. Generate and paste

1. Click **Generate token**
2. Copy the token and paste it below
3. If your organization uses SAML SSO, authorize this token for that organization

If you must use a classic PAT, use at least these scopes:

~~~text
repo
workflow
read:org
~~~
`

var resourceLabelsByType = map[string]string{
	ResourceIssues:                "Issues",
	ResourcePullRequests:          "Pull Requests",
	ResourceWorkflows:             "Workflows (GitHub Actions)",
	ResourceReleases:              "Releases",
	ResourceCode:                  "Code (Pushes, Branches, Tags, Commit Status)",
	ResourceRepositoryPermissions: "Repository Access / Permissions",
}

var patPermissionLinesByResourceType = map[string][]string{
	ResourceIssues: {
		"- Issues: Read and write",
	},
	ResourcePullRequests: {
		"- Pull requests: Read and write",
	},
	ResourceWorkflows: {
		"- Actions: Read and write",
	},
	ResourceReleases: {
		"- Contents: Read and write",
	},
	ResourceCode: {
		"- Contents: Read-only",
		"- Commit statuses: Read and write",
	},
	ResourceRepositoryPermissions: {
		"- Administration: Read-only",
	},
}

func (g *SetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	return core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectOwner,
		Label: "Which GitHub user or organization do you want to connect?",
		Inputs: []configuration.Field{
			{
				Name:        ParameterOwner,
				Label:       "GitHub user or organization",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Placeholder: "e.g. superplanehq",
			},
		},
		Instructions: ownerInstructions,
	}
}

func (g *SetupProvider) OnStepSubmit(stepName string, inputs any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch stepName {
	case SetupStepSelectOwner:
		return g.onSelectOwnerSubmit(inputs, ctx)
	case SetupStepSelectResources:
		return g.onSelectResourcesSubmit(inputs, ctx)
	case SetupStepSelectAuthMethod:
		return g.onSelectAuthMethodSubmit(inputs, ctx)
	case SetupStepEnterPAT:
		return g.onEnterPATSubmit(inputs, ctx)
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
		Name:        ParameterOwner,
		Label:       "Owner",
		Description: "GitHub user or organization",
		Type:        configuration.FieldTypeString,
		Value:       owner,
		Editable:    false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectResources,
		Label: "Which GitHub resources do you want to access?",
		Inputs: []configuration.Field{
			{
				Name:     ParameterResources,
				Label:    "Resources",
				Type:     configuration.FieldTypeMultiSelect,
				Required: true,
				Default:  AllResourceTypes,
				TypeOptions: &configuration.TypeOptions{
					MultiSelect: &configuration.MultiSelectTypeOptions{Options: ResourceTypeOptions},
				},
			},
		},
		Instructions: resourcesInstructions,
	}, nil
}

func (g *SetupProvider) onSelectResourcesSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	resources, err := parseResourceTypes(m[ParameterResources])
	if err != nil {
		return nil, err
	}

	err = ctx.Parameters.Create(core.IntegrationParameterDefinition{
		Name:        ParameterResources,
		Label:       "Resource Types",
		Description: "Selected GitHub resource groups",
		Type:        configuration.FieldTypeMultiSelect,
		Value:       resources,
		Editable:    false,
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
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "Personal Access Token", Value: AuthMethodPAT},
							{Label: "Private GitHub App", Value: AuthMethodGitHubApp},
						},
					},
				},
			},
		},
		Instructions: authMethodInstructions,
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
		Name:        ParameterAuthMethod,
		Label:       "Authentication Method",
		Description: "GitHub authentication method",
		Type:        configuration.FieldTypeString,
		Value:       authMethod,
		Editable:    false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	switch authMethod {
	case AuthMethodPAT:
		owner, err := getStringParameter(ctx.Parameters, ParameterOwner)
		if err != nil {
			return nil, err
		}

		resources, err := getResourceTypesFromParameters(ctx)
		if err != nil {
			return nil, err
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
			Instructions: buildPATInstructions(owner, resources),
		}, nil

	case AuthMethodGitHubApp:
		return nil, fmt.Errorf("TODO: implement GitHub app authentication")

	default:
		return nil, fmt.Errorf("not implemented")
	}
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
		Editable: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	owner, err := getStringParameter(ctx.Parameters, ParameterOwner)
	if err != nil {
		return nil, err
	}

	if err := validatePATConnection(ctx.HTTP, token, owner); err != nil {
		return nil, err
	}

	if err := registerCapabilities(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

func parseResourceTypes(value any) ([]string, error) {
	resources, err := toStringSlice(value)
	if err != nil {
		return nil, fmt.Errorf("invalid resource types")
	}

	if len(resources) == 0 {
		return nil, errors.New("at least one resource type is required")
	}

	allowed := map[string]bool{}
	for _, resourceType := range AllResourceTypes {
		allowed[resourceType] = true
	}

	seen := map[string]bool{}
	result := []string{}
	for _, resourceType := range resources {
		if !allowed[resourceType] {
			return nil, fmt.Errorf("invalid resource type: %s", resourceType)
		}
		if seen[resourceType] {
			continue
		}
		seen[resourceType] = true
		result = append(result, resourceType)
	}

	sort.Strings(result)
	return result, nil
}

func toStringSlice(v any) ([]string, error) {
	switch values := v.(type) {
	case []string:
		return values, nil
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("invalid string value")
			}
			result = append(result, s)
		}
		return result, nil
	default:
		return nil, errors.New("invalid string array")
	}
}

func getStringParameter(parameters core.IntegrationParameterStorage, name string) (string, error) {
	value, err := parameters.Get(name)
	if err != nil {
		return "", fmt.Errorf("error getting parameter %s: %v", name, err)
	}

	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid parameter %s", name)
	}

	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("parameter %s is required", name)
	}

	return s, nil
}

func getResourceTypesFromParameters(ctx core.SetupStepContext) ([]string, error) {
	value, err := ctx.Parameters.Get(ParameterResources)
	if err != nil {
		return nil, fmt.Errorf("error getting resource types: %v", err)
	}

	return parseResourceTypes(value)
}

func buildPATInstructions(owner string, resources []string) string {
	selectedResourceLines := make([]string, 0, len(resources))
	requiredPermissionLines := []string{"- Metadata: Read-only (required by GitHub)"}

	seenPermissions := map[string]bool{
		"- Metadata: Read-only (required by GitHub)": true,
	}
	selectedSet := map[string]bool{}
	for _, resourceType := range resources {
		selectedSet[resourceType] = true
	}

	for _, resourceType := range AllResourceTypes {
		if !selectedSet[resourceType] {
			continue
		}

		label, ok := resourceLabelsByType[resourceType]
		if !ok {
			continue
		}

		selectedResourceLines = append(selectedResourceLines, "- "+label)

		for _, permissionLine := range patPermissionLinesByResourceType[resourceType] {
			if seenPermissions[permissionLine] {
				continue
			}
			seenPermissions[permissionLine] = true
			requiredPermissionLines = append(requiredPermissionLines, permissionLine)
		}
	}

	return fmt.Sprintf(
		patInstructionsTemplate,
		owner,
		strings.Join(selectedResourceLines, "\n"),
		strings.Join(requiredPermissionLines, "\n"),
	)
}

func registerCapabilities(ctx core.SetupStepContext) error {
	resources, err := getResourceTypesFromParameters(ctx)
	if err != nil {
		return err
	}

	componentsByName := map[string]core.Component{}
	triggersByName := map[string]core.Trigger{}

	for _, resourceType := range resources {
		for _, component := range componentsByResourceType(resourceType) {
			componentsByName[component.Name()] = component
		}

		for _, trigger := range triggersByResourceType(resourceType) {
			triggersByName[trigger.Name()] = trigger
		}
	}

	components := make([]core.Component, 0, len(componentsByName))
	for _, component := range componentsByName {
		components = append(components, component)
	}

	triggers := make([]core.Trigger, 0, len(triggersByName))
	for _, trigger := range triggersByName {
		triggers = append(triggers, trigger)
	}

	if err := ctx.Capabilities.RegisterComponents(components); err != nil {
		return fmt.Errorf("error registering components: %v", err)
	}

	if err := ctx.Capabilities.RegisterTriggers(triggers); err != nil {
		return fmt.Errorf("error registering triggers: %v", err)
	}

	return nil
}

func componentsByResourceType(resourceType string) []core.Component {
	switch resourceType {
	case ResourceIssues:
		return []core.Component{
			&GetIssue{},
			&CreateIssue{},
			&UpdateIssue{},
			&AddIssueLabel{},
			&RemoveIssueLabel{},
			&AddIssueAssignee{},
			&RemoveIssueAssignee{},
			&CreateIssueComment{},
		}
	case ResourcePullRequests:
		return []core.Component{
			&CreateReview{},
			&AddReaction{},
			&CreateIssueComment{},
		}
	case ResourceWorkflows:
		return []core.Component{
			&RunWorkflow{},
			&GetWorkflowUsage{},
		}
	case ResourceReleases:
		return []core.Component{
			&CreateRelease{},
			&GetRelease{},
			&UpdateRelease{},
			&DeleteRelease{},
		}
	case ResourceCode:
		return []core.Component{
			&PublishCommitStatus{},
		}
	case ResourceRepositoryPermissions:
		return []core.Component{
			&GetRepositoryPermission{},
		}
	default:
		return []core.Component{}
	}
}

func triggersByResourceType(resourceType string) []core.Trigger {
	switch resourceType {
	case ResourceIssues:
		return []core.Trigger{
			&OnIssue{},
			&OnIssueComment{},
		}
	case ResourcePullRequests:
		return []core.Trigger{
			&OnPullRequest{},
			&OnPRComment{},
			&OnPRReviewComment{},
		}
	case ResourceWorkflows:
		return []core.Trigger{
			&OnWorkflowRun{},
		}
	case ResourceReleases:
		return []core.Trigger{
			&OnRelease{},
		}
	case ResourceCode:
		return []core.Trigger{
			&OnPush{},
			&OnBranchCreated{},
			&OnTagCreated{},
		}
	case ResourceRepositoryPermissions:
		return []core.Trigger{}
	default:
		return []core.Trigger{}
	}
}

func validatePATConnection(_ core.HTTPContext, token string, owner string) error {
	client := gh.NewClient(
		oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		),
	)

	_, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return fmt.Errorf("failed to validate GitHub personal access token: %v", err)
	}

	// Validate access to the selected owner by trying org repos first,
	// then falling back to user repos.
	_, orgResp, orgErr := client.Repositories.ListByOrg(context.Background(), owner, &gh.RepositoryListByOrgOptions{ListOptions: gh.ListOptions{PerPage: 1}})
	if orgErr == nil {
		_ = orgResp
		return nil
	}

	_, userResp, userErr := client.Repositories.ListByUser(context.Background(), owner, &gh.RepositoryListByUserOptions{ListOptions: gh.ListOptions{PerPage: 1}})
	if userErr == nil {
		_ = userResp
		return nil
	}

	return fmt.Errorf("failed to access repositories for owner %s", owner)
}
