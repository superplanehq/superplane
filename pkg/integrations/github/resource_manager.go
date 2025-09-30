package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	URL    string
	Owner  string
}

func NewGitHubResourceManager(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.ResourceManager, error) {
	owner, err := parseOwnerFromURL(URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	token, err := authenticate()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %v", err)
	}

	return &GitHubResourceManager{
		URL:   URL,
		Owner: owner,
		client: github.NewClient(
			oauth2.NewClient(ctx,
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
			),
		)}, nil
}

// URL should be https://github.com/<owner>
func parseOwnerFromURL(URL string) (string, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL %s: %v", URL, err)
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("%s does not use HTTPS", URL)
	}

	if u.Host != "github.com" {
		return "", fmt.Errorf("URL %s is not a GitHub URL", URL)
	}

	return u.Path[1:], nil
}

func (i *GitHubResourceManager) SetupWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	//
	// If a webhook already exists for this repository,
	// we update it. If not, we create a new one.
	//
	webhookIndex := slices.IndexFunc(options.Children, func(r integrations.Resource) bool {
		return r.Type() == ResourceTypeWebHook
	})

	if webhookIndex == -1 {
		return i.createRepositoryWebhook(options)
	}

	hook := options.Children[webhookIndex]
	return i.updateRepositoryWebhook(hook, options)
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

func (i *GitHubResourceManager) createRepositoryWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: i.getEventTypes(options),
		Config: &github.HookConfig{
			URL:         &options.URL,
			Secret:      github.Ptr(string(options.Key)),
			ContentType: github.Ptr("json"),
		},
	}

	createdHook, _, err := i.client.Repositories.CreateHook(context.Background(), i.Owner, options.Parent.Name(), hook)
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

func (i *GitHubResourceManager) updateRepositoryWebhook(hook integrations.Resource, options integrations.WebhookOptions) ([]integrations.Resource, error) {
	hookID, err := strconv.ParseInt(hook.Id(), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook ID: %v", err)
	}

	updatedHook, _, err := i.client.Repositories.EditHook(context.Background(), i.Owner, options.Parent.Name(), hookID, &github.Hook{
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

func (i *GitHubResourceManager) CleanupWebhook(parentResource integrations.Resource, webhook integrations.Resource) error {
	hookID, err := strconv.ParseInt(webhook.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing webhook ID: %v", err)
	}

	_, err = i.client.Repositories.DeleteHook(context.Background(), i.Owner, parentResource.Name(), hookID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

func (i *GitHubResourceManager) List(ctx context.Context, resourceType string) ([]integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return i.listRepositories(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
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

func (i *GitHubResourceManager) Cancel(resourceType, id string, parentResource integrations.Resource) error {
	switch resourceType {
	case ResourceTypeWorkflow:
		return i.stopWorkflowRun(parentResource, id)
	default:
		return fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubResourceManager) stopWorkflowRun(parentResource integrations.Resource, id string) error {
	runID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	//
	// GitHub SDK returns an error even though it got a 202 response back :)
	//
	response, err := i.client.Actions.CancelWorkflowRunByID(context.Background(), i.Owner, parentResource.Name(), runID)
	if response.StatusCode == http.StatusAccepted {
		return nil
	}

	return fmt.Errorf("Cancel request for %s received status code %d: %v", id, response.StatusCode, err)
}

func (i *GitHubResourceManager) getRepository(repoName string) (integrations.Resource, error) {
	repository, _, err := i.client.Repositories.Get(context.Background(), i.Owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	return &Repository{
		ID:             repository.GetID(),
		RepositoryName: repository.GetName(),
		HTMLURL:        repository.GetHTMLURL(),
	}, nil
}

func (i *GitHubResourceManager) listRepositories(ctx context.Context) ([]integrations.Resource, error) {
	repositories, _, err := i.client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	values := make([]integrations.Resource, len(repositories))
	for _, repository := range repositories {
		values = append(values, &Repository{
			ID:             repository.GetID(),
			RepositoryName: repository.GetName(),
			HTMLURL:        repository.GetHTMLURL(),
		})
	}

	return values, nil
}

func (i *GitHubResourceManager) getWorkflowRun(parentResource integrations.Resource, id string) (integrations.StatefulResource, error) {
	runID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}

	workflowRun, _, err := i.client.Actions.GetWorkflowRunByID(context.Background(), i.Owner, parentResource.Name(), runID)
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
	ID             int64
	RepositoryName string
	HTMLURL        string
}

func (r *Repository) Id() string {
	return fmt.Sprintf("%d", r.ID)
}

func (r *Repository) Name() string {
	return r.RepositoryName
}

func (r *Repository) Type() string {
	return ResourceTypeRepository
}

func (r *Repository) URL() string {
	return r.HTMLURL
}

type WorkflowRun struct {
	ID         int64
	Status     string
	Conclusion string
	HtmlUTL    string
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

func (w *WorkflowRun) URL() string {
	return w.HtmlUTL
}
