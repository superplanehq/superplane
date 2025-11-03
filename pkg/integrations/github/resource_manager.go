package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
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

type WebhookConfiguration struct {
	Events []string `json:"events"`
}

func (i *GitHubResourceManager) SetupWebhook(options integrations.WebhookOptions) (any, error) {
	config := WebhookConfiguration{}
	err := mapstructure.Decode(options.Configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: config.Events,
		Config: &github.HookConfig{
			URL:         &options.URL,
			Secret:      github.Ptr(string(options.Secret)),
			ContentType: github.Ptr("json"),
		},
	}

	createdHook, _, err := i.client.Repositories.CreateHook(context.Background(), i.Owner, options.Resource.Name(), hook)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &Webhook{
		ID:          createdHook.GetID(),
		WebhookName: *createdHook.Name,
		WebhookURL:  createdHook.GetURL(),
	}, nil
}

func (i *GitHubResourceManager) CleanupWebhook(options integrations.WebhookOptions) error {
	webhook := &Webhook{}
	err := mapstructure.Decode(options.Metadata, &webhook)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	hookID, err := strconv.ParseInt(webhook.Id(), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing webhook ID: %v", err)
	}

	_, err = i.client.Repositories.DeleteHook(context.Background(), i.Owner, options.Resource.Name(), hookID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

func (i *GitHubResourceManager) Get(resourceType, id string) (integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return i.getRepository(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubResourceManager) List(resourceType string) ([]integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return i.listRepositories()
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubResourceManager) listRepositories() ([]integrations.Resource, error) {
	_, _, err := i.client.Organizations.Get(context.Background(), i.Owner)
	if err == nil {
		return i.listOrganizationRepositories()
	}

	opts := &github.RepositoryListByUserOptions{}
	repositories, _, err := i.client.Repositories.ListByUser(context.Background(), i.Owner, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	resources := []integrations.Resource{}
	for _, repository := range repositories {
		resources = append(resources, &Repository{
			ID:             repository.GetID(),
			RepositoryName: repository.GetFullName(),
			RepositoryURL:  repository.GetHTMLURL(),
		})
	}

	return resources, nil
}

func (i *GitHubResourceManager) listOrganizationRepositories() ([]integrations.Resource, error) {
	opts := &github.RepositoryListByOrgOptions{}
	repositories, _, err := i.client.Repositories.ListByOrg(context.Background(), i.Owner, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	resources := []integrations.Resource{}
	for _, repository := range repositories {
		resources = append(resources, &Repository{
			ID:             repository.GetID(),
			RepositoryName: repository.GetFullName(),
			RepositoryURL:  repository.GetHTMLURL(),
		})
	}

	return resources, nil
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

func (i *GitHubResourceManager) getRepository(idOrName string) (integrations.Resource, error) {
	id, err := strconv.ParseInt(idOrName, 10, 64)
	if err == nil {
		return i.getRepositoryByID(id)
	}

	repository, _, err := i.client.Repositories.Get(context.Background(), i.Owner, idOrName)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	return &Repository{
		ID:             repository.GetID(),
		RepositoryName: repository.GetName(),
		RepositoryURL:  repository.GetHTMLURL(),
	}, nil
}

func (i *GitHubResourceManager) getRepositoryByID(id int64) (integrations.Resource, error) {
	repository, _, err := i.client.Repositories.GetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %v", err)
	}

	return &Repository{
		ID:             repository.GetID(),
		RepositoryName: repository.GetName(),
		RepositoryURL:  repository.GetHTMLURL(),
	}, nil
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
	RepositoryURL  string
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
	return r.RepositoryURL
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
