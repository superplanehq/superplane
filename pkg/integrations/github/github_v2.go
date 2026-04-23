package github

import (
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GithubV2 struct{}

func (g *GithubV2) Name() string {
	return "github"
}

func (g *GithubV2) Label() string {
	return "GitHub"
}

func (g *GithubV2) Description() string {
	return "Manage and react to changes in your GitHub repositories"
}

func (g *GithubV2) FirstStep() core.SetupStep {
	//
	// TODO: is this really the first step?
	// Shouldn't it be the user/org selection?
	//

	return core.SetupStep{
		Name:         "selectConnectionMode",
		Instructions: "Select the connection mode you want to use",
		Inputs: []configuration.Field{
			{
				Name:     "connectionMode",
				Label:    "Connection Mode",
				Type:     configuration.FieldTypeSelect,
				Required: true,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "GitHub App", Value: "github_app"},
							{Label: "Personal Access Token", Value: "pat"},
						},
					},
				},
			},
		},
	}
}

func (g *GithubV2) OnStepSubmit(stepName string, input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch stepName {
	case "selectConnectionMode":
		return g.onSelectConnectionModeSubmit(input, ctx)
	case "enterPersonalAccessToken":
		return g.onEnterPersonalAccessTokenSubmit(input, ctx)
	case "setupGitHubApp":
		//
		// TODO: not sure how this looks like, since the app creation + installation
		// is done through HTTP redirects with GitHub.
		//
		return nil, nil
	}

	return nil, errors.New("unknown step")
}

func (g *GithubV2) onSelectConnectionModeSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	connectionMode, ok := m["connectionMode"].(string)
	if !ok {
		return nil, errors.New("invalid connection mode")
	}

	if connectionMode != "github_app" && connectionMode != "pat" {
		return nil, errors.New("invalid connection mode")
	}

	//
	// Connection mode is not something you can change.
	//
	err := ctx.Parameters.Create("connectionMode", core.IntegrationParameterDefinition{
		Type:     configuration.FieldTypeString,
		Value:    connectionMode,
		Editable: false,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating parameter: %v", err)
	}

	//
	// Find next steps
	//
	switch connectionMode {
	case "github_app":

		//
		// TODO
		//
		return &core.SetupStep{
			Type:         core.SetupStepTypeRedirectPrompt,
			Name:         "setupGitHubApp",
			Instructions: "Install the GitHub app",
			RedirectPrompt: &core.RedirectPrompt{
				URL:      "????",
				Method:   "POST",
				FormData: map[string]string{},
			},
		}, nil

	case "pat":
		return &core.SetupStep{
			Name:         "enterPersonalAccessToken",
			Instructions: "Enter your personal access token",
			Inputs: []configuration.Field{
				{
					Name:     "personalAccessToken",
					Label:    "Personal Access Token",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
			},
		}, nil

	default:
		return nil, errors.New("invalid connection mode")
	}
}

func (g *GithubV2) onEnterPersonalAccessTokenSubmit(input any, ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := input.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	personalAccessToken, ok := m["personalAccessToken"].(string)
	if !ok {
		return nil, errors.New("invalid personal access token")
	}

	if personalAccessToken == "" {
		return nil, errors.New("personal access token is required")
	}

	//
	// Store the personal access token as an editable secret.
	//
	err := ctx.Secrets.Create("personalAccessToken", core.IntegrationSecretDefinition{
		Value:    []byte(personalAccessToken),
		Editable: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	err = ctx.Capabilities.RegisterComponents([]core.Component{
		&GetIssue{},
		&GetRepositoryPermission{},
		&CreateIssue{},
		&CreateIssueComment{},
		&AddReaction{},
		&UpdateIssue{},
		&AddIssueLabel{},
		&RemoveIssueLabel{},
		&AddIssueAssignee{},
		&RemoveIssueAssignee{},
		&CreateReview{},
		&RunWorkflow{},
		&PublishCommitStatus{},
		&CreateRelease{},
		&GetRelease{},
		&UpdateRelease{},
		&DeleteRelease{},
		&GetWorkflowUsage{},
	})

	if err != nil {
		return nil, fmt.Errorf("error registering components: %v", err)
	}

	err = ctx.Capabilities.RegisterTriggers([]core.Trigger{
		&OnPush{},
		&OnPullRequest{},
		&OnPRComment{},
		&OnPRReviewComment{},
		&OnIssue{},
		&OnIssueComment{},
		&OnRelease{},
		&OnTagCreated{},
		&OnBranchCreated{},
		&OnWorkflowRun{},
	})

	if err != nil {
		return nil, fmt.Errorf("error registering triggers: %v", err)
	}

	return nil, nil
}
