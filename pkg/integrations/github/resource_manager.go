package github

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/integrations"
	"golang.org/x/oauth2"
)

var defaultEventTypes = []string{
	"create",
	"package",
	"pull_request",
	"push",
	"registry_package",
	"release",
	"workflow_run",
}

type GitHubResourceManager struct {
	client *github.Client
}

func NewGitHubResourceManager(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.ResourceManager, error) {
	//
	// TODO: figure out if the URL will be important here or not.
	//

	token, err := authenticate()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %v", err)
	}

	return &GitHubResourceManager{
		client: github.NewClient(
			oauth2.NewClient(ctx,
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
			),
		)}, nil
}

func (i *GitHubResourceManager) SetupWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	owner, repo, err := parseRepoName(options.Parent.Name())
	if err != nil {
		return nil, err
	}

	//
	// If a webhook already exists for this repository,
	// we update it. If not, we create a new one.
	//
	webhookIndex := slices.IndexFunc(options.Children, func(r integrations.Resource) bool {
		return r.Type() == ResourceTypeWebHook
	})

	if webhookIndex == -1 {
		return i.createRepositoryWebhook(owner, repo, options)
	}

	hook := options.Children[webhookIndex]
	return i.updateRepositoryWebhook(owner, repo, hook, options)
}

func (i *GitHubResourceManager) getEventTypes(options integrations.WebhookOptions) []string {
	//
	// If we are creating a webhook for an internally-scoped event source,
	// we only need to listen to workflow_run events.
	//
	if options.Internal {
		return []string{"workflow_run"}
	}

	//
	// If no event types are selected by the user,
	// we use a sensible set of default event types.
	//
	if len(options.EventTypes) == 0 {
		return defaultEventTypes
	}

	//
	// We always include the workflow_run type of event,
	// to ensure that we can re-use the same webhook for stage execution updates.
	//
	if !slices.Contains(options.EventTypes, "workflow_run") {
		return append(options.EventTypes, "workflow_run")
	}

	return options.EventTypes
}

func (i *GitHubResourceManager) createRepositoryWebhook(owner, repo string, options integrations.WebhookOptions) ([]integrations.Resource, error) {
	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: i.getEventTypes(options),
		Config: &github.HookConfig{
			URL:         &options.URL,
			Secret:      github.Ptr(string(options.Key)),
			ContentType: github.Ptr("json"),
		},
	}

	createdHook, _, err := i.client.Repositories.CreateHook(context.Background(), owner, repo, hook)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return []integrations.Resource{
		&Webhook{
			ID:          createdHook.GetID(),
			WebhookName: *createdHook.Name,
		},
	}, nil
}

func (i *GitHubResourceManager) updateRepositoryWebhook(owner, repo string, hook integrations.Resource, options integrations.WebhookOptions) ([]integrations.Resource, error) {
	hookID, err := strconv.ParseInt(hook.Id(), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook ID: %v", err)
	}

	updatedHook, _, err := i.client.Repositories.EditHook(context.Background(), owner, repo, hookID, &github.Hook{
		Active: github.Ptr(true),
		Events: i.getEventTypes(options),
		Config: &github.HookConfig{
			URL:         &options.URL,
			Secret:      github.Ptr(string(options.Key)),
			ContentType: github.Ptr("json"),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error updating webhook: %v", err)
	}

	return []integrations.Resource{
		&Webhook{
			ID:          updatedHook.GetID(),
			WebhookName: *updatedHook.Name,
		},
	}, nil
}

func (i *GitHubResourceManager) Get(resourceType, id string) (integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return i.getRepository(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubResourceManager) Status(resourceType, id string, parentResource integrations.Resource) (integrations.StatefulResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		return i.getWorkflowRun(parentResource, id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubResourceManager) getRepository(fullName string) (integrations.Resource, error) {
	owner, repo, err := parseRepoName(fullName)
	if err != nil {
		return nil, err
	}

	repository, _, err := i.client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	return &Repository{
		ID:       repository.GetID(),
		FullName: repository.GetFullName(),
	}, nil
}

func (i *GitHubResourceManager) getWorkflowRun(parentResource integrations.Resource, id string) (integrations.StatefulResource, error) {
	runID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}

	owner, repo, err := parseRepoName(parentResource.Name())
	if err != nil {
		return nil, err
	}

	workflowRun, _, err := i.client.Actions.GetWorkflowRunByID(context.Background(), owner, repo, runID)
	if err != nil {
		return nil, fmt.Errorf("error getting workflow run: %v", err)
	}

	return &WorkflowRun{
		ID:         workflowRun.GetID(),
		Status:     workflowRun.GetStatus(),
		Conclusion: workflowRun.GetConclusion(),
	}, nil
}

type Repository struct {
	ID       int64
	FullName string
}

func (r *Repository) Id() string {
	return fmt.Sprintf("%d", r.ID)
}

func (r *Repository) Name() string {
	return r.FullName
}

func (r *Repository) Type() string {
	return ResourceTypeRepository
}

type WorkflowRun struct {
	ID         int64
	Status     string
	Conclusion string
}

func (w *WorkflowRun) Id() string {
	return fmt.Sprintf("%d", w.ID)
}

func (w *WorkflowRun) Type() string {
	return ResourceTypeWorkflow
}

func (w *WorkflowRun) Finished() bool {
	return w.Status == "completed"
}

func (w *WorkflowRun) Successful() bool {
	return w.Conclusion == "success"
}
