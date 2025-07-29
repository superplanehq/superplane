package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/integrations"
	"golang.org/x/oauth2"
)

const (
	ResourceTypeRepository = "repository"
	ResourceTypeWorkflow   = "workflow"
	ResourceTypeWebHook    = "webhook"
)

type GitHubIntegration struct {
	client *github.Client
	token  string
}

// TODO: not sure about the URL here
func NewGitHubIntegration(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.Integration, error) {
	token, err := authenticate()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication: %v", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GitHubIntegration{
		client: client,
		token:  token,
	}, nil
}

func (i *GitHubIntegration) Get(resourceType, id string) (integrations.Resource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return i.getRepository(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubIntegration) Check(resourceType, id string) (integrations.StatefulResource, error) {
	switch resourceType {
	case ResourceTypeWorkflow:
		return i.getWorkflowRun(id)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resourceType)
	}
}

func (i *GitHubIntegration) SetupWebhook(options integrations.WebhookOptions) ([]integrations.Resource, error) {
	owner, repo, err := parseRepoName(options.Resource.Name())
	if err != nil {
		return nil, err
	}

	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: []string{"push", "workflow_run"},
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

func (i *GitHubIntegration) HandleWebhook(payload []byte) (integrations.StatefulResource, error) {
	//
	// TODO: I'm hardcoding event type here, but we also listen to different event types
	// and we should make sure we filter them out somehow.
	//
	event, err := github.ParseWebHook("workflow_run", payload)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook: %v", err)
	}

	switch e := event.(type) {
	case *github.WorkflowRunEvent:
		return &WorkflowRun{
			ID:         e.GetWorkflowRun().GetID(),
			Status:     e.GetWorkflowRun().GetStatus(),
			Conclusion: e.GetWorkflowRun().GetConclusion(),
			Repository: e.GetRepo().GetFullName(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported event type %T", event)
	}
}

func (i *GitHubIntegration) getRepository(fullName string) (integrations.Resource, error) {
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

func (i *GitHubIntegration) getWorkflowRun(id string) (integrations.StatefulResource, error) {
	repository, runID, err := parseWorkflowRunID(id)
	if err != nil {
		return nil, err
	}

	owner, repo, err := parseRepoName(repository)
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
		Repository: repository,
	}, nil
}

func parseRepoName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid repository name format: %s", fullName)
}

func parseWorkflowRunID(id string) (string, int64, error) {
	parts := strings.Split(id, ":")
	if len(parts) == 2 {
		runID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return "", -1, fmt.Errorf("invalid run ID: %s", parts[1])
		}

		return parts[0], runID, nil
	}

	return "", -1, fmt.Errorf("invalid workflow run ID format: %s (expected format: owner/repo:runID)", id)
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
	Repository string
}

func (w *WorkflowRun) Id() string {
	return fmt.Sprintf("%s:%d", w.Repository, w.ID)
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

type Webhook struct {
	ID          int64
	WebhookName string
	Repo        *Repository
}

func (h *Webhook) Id() string {
	return fmt.Sprintf("%d", h.ID)
}

func (h *Webhook) Name() string {
	return h.WebhookName
}

func (h *Webhook) Type() string {
	return ResourceTypeWebHook
}
