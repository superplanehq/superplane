package factories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

var (
	ErrTaskPullRequestGitHubOnly         = errors.New("pull requests open only in GitHub workspaces")
	ErrTaskPullRequestGitHubNotConnected = errors.New("the workspace GitHub integration is not connected")
	ErrTaskPullRequestOtherTask          = errors.New("the pull request belongs to another task")
)

// TaskPullRequestInputError is an invalid open request. Its message is safe
// to show to the agent.
type TaskPullRequestInputError struct {
	message string
}

func (e *TaskPullRequestInputError) Error() string {
	return e.message
}

func invalidTaskPullRequest(message string) error {
	return &TaskPullRequestInputError{message: message}
}

// TaskPullRequestInput describes a pushed branch that needs a pull request.
type TaskPullRequestInput struct {
	Repository string
	Head       string
	Base       string
	Title      string
	Body       string
	Draft      bool
	Run        *factoryevents.RunRef
}

type taskPullRequestGitHubAPI interface {
	FindRepository(repository string) (*github.Repository, error)
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	CreatePullRequest(ctx context.Context, repository string, pullRequest *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	EditPullRequest(ctx context.Context, repository string, pullNumber int, pullRequest *github.PullRequest) (*github.PullRequest, *github.Response, error)
}

var newTaskPullRequestGitHubAPI = func(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (taskPullRequestGitHubAPI, error) {
	return githubIntegrationClient(db, deps, factory)
}

// OpenTaskPullRequest opens the pull request of a pushed branch in a
// workspace repository, or updates the open one, and attaches it to the task.
// A second call for the same repository and branch updates the same pull
// request.
func OpenTaskPullRequest(
	ctx context.Context,
	deps IntakeDependencies,
	baseURL string,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (*models.FactoryPullRequest, error) {
	db := database.DB(ctx)
	input, err := normalizeTaskPullRequestInput(db, factory, order, input)
	if err != nil {
		return nil, err
	}

	client, err := newTaskPullRequestGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, ErrTaskPullRequestGitHubNotConnected
	}
	if input.Base == "" {
		repository, err := client.FindRepository(input.Repository)
		if err != nil {
			return nil, err
		}
		input.Base = repository.GetDefaultBranch()
	}

	open, err := findOpenTaskPullRequest(ctx, client, input)
	if err != nil {
		return nil, err
	}
	if open != nil {
		if err := requireTaskOwnsPullRequest(db, factory, order, input.Repository, open); err != nil {
			return nil, err
		}
	}
	remote, err := createOrUpdateTaskPullRequest(ctx, client, input, open, taskPullRequestBody(baseURL, factory, order, input.Body))
	if err != nil {
		return nil, err
	}

	var pullRequest *models.FactoryPullRequest
	created := false
	err = db.Transaction(func(tx *gorm.DB) error {
		pullRequest, created, err = attachTaskPullRequest(tx, factory, order, input, remote)
		return err
	})
	if err != nil {
		return nil, err
	}

	if created {
		startPullRequestFollowUps(ctx, deps, factory, pullRequest)
	}
	reason := factoryevents.EventTypeOrderPullRequestUpdated
	if created {
		reason = factoryevents.EventTypeOrderPullRequestAdded
	}
	if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), order.ID.String(), reason); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	return pullRequest, nil
}

// ListTaskPullRequests returns the pull requests on the task, oldest first.
func ListTaskPullRequests(tx *gorm.DB, order *models.FactoryWorkOrder) ([]models.FactoryPullRequest, error) {
	return trackedPullRequests(tx, order.ID)
}

func normalizeTaskPullRequestInput(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (TaskPullRequestInput, error) {
	if factory.OnboardingConfigValue().EffectiveVCSProvider() != models.ProviderGitHub {
		return input, ErrTaskPullRequestGitHubOnly
	}
	input.Repository = strings.TrimSpace(input.Repository)
	input.Head = strings.TrimPrefix(strings.TrimSpace(input.Head), "refs/heads/")
	input.Base = strings.TrimPrefix(strings.TrimSpace(input.Base), "refs/heads/")
	input.Title = strings.TrimSpace(input.Title)
	if input.Repository == "" {
		return input, invalidTaskPullRequest("repository is required")
	}
	if input.Head == "" {
		return input, invalidTaskPullRequest("head is required")
	}
	if input.Title == "" {
		return input, invalidTaskPullRequest("title is required")
	}

	inWorkspace, err := factory.IsTaskRepository(tx, order, input.Repository)
	if err != nil {
		return input, err
	}
	if !inWorkspace {
		return input, models.ErrFactoryPullRequestRepositoryNotInWorkspace
	}
	if input.Base == "" && order.Repository != nil && strings.EqualFold(strings.TrimSpace(*order.Repository), input.Repository) && order.DefaultBranch != nil {
		input.Base = strings.TrimSpace(*order.DefaultBranch)
	}
	if input.Base != "" && input.Base == input.Head {
		return input, invalidTaskPullRequest("head and base must be different branches")
	}
	return input, nil
}

func findOpenTaskPullRequest(ctx context.Context, client taskPullRequestGitHubAPI, input TaskPullRequestInput) (*github.PullRequest, error) {
	owner, _, _ := strings.Cut(input.Repository, "/")
	open, _, err := client.ListPullRequests(ctx, input.Repository, &github.PullRequestListOptions{
		State: "open",
		Head:  owner + ":" + input.Head,
		Base:  input.Base,
	})
	if err != nil || len(open) == 0 {
		return nil, err
	}
	return open[0], nil
}

// requireTaskOwnsPullRequest rejects an open pull request that another task
// tracks. The check runs before the GitHub edit, which cannot be undone.
func requireTaskOwnsPullRequest(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	repository string,
	open *github.PullRequest,
) error {
	existing, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: taskPullRequestRepository(repository, open),
		Number:     int64(open.GetNumber()),
	})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.WorkOrderID != order.ID {
		return ErrTaskPullRequestOtherTask
	}
	return nil
}

func taskPullRequestRepository(repository string, remote *github.PullRequest) string {
	if fullName := remote.GetBase().GetRepo().GetFullName(); fullName != "" {
		return fullName
	}
	return repository
}

func createOrUpdateTaskPullRequest(
	ctx context.Context,
	client taskPullRequestGitHubAPI,
	input TaskPullRequestInput,
	open *github.PullRequest,
	body string,
) (*github.PullRequest, error) {
	if open != nil {
		updated, _, err := client.EditPullRequest(ctx, input.Repository, open.GetNumber(), &github.PullRequest{
			Title: github.Ptr(input.Title),
			Body:  github.Ptr(body),
		})
		return updated, err
	}

	created, _, err := client.CreatePullRequest(ctx, input.Repository, &github.NewPullRequest{
		Title: github.Ptr(input.Title),
		Head:  github.Ptr(input.Head),
		Base:  github.Ptr(input.Base),
		Body:  github.Ptr(body),
		Draft: github.Ptr(input.Draft),
	})
	return created, err
}

// attachTaskPullRequest records the pull request and its branch on the task.
// It reports created=false when the task already has the pull request.
func attachTaskPullRequest(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
	remote *github.PullRequest,
) (*models.FactoryPullRequest, bool, error) {
	state := models.FactoryPullRequestStateOpen
	if remote.GetDraft() {
		state = models.FactoryPullRequestStateDraft
	}
	repository := taskPullRequestRepository(input.Repository, remote)
	title := remote.GetTitle()
	url := remote.GetHTMLURL()

	existing, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: repository,
		Number:     int64(remote.GetNumber()),
	})
	if err != nil && !errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return nil, false, err
	}

	var pullRequest *models.FactoryPullRequest
	created := false
	switch {
	case existing != nil && existing.WorkOrderID != order.ID:
		return nil, false, ErrTaskPullRequestOtherTask
	case existing != nil:
		if err := existing.Update(tx, models.FactoryPullRequestPatch{
			URL:   &url,
			Title: &title,
			State: &state,
			Run:   input.Run,
		}); err != nil {
			return nil, false, err
		}
		pullRequest = existing
	default:
		pullRequest, err = order.CreatePullRequest(tx, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			ExternalID: strconv.FormatInt(remote.GetID(), 10),
			Repository: repository,
			Number:     int64(remote.GetNumber()),
			URL:        url,
			Title:      title,
			State:      state,
			Run:        input.Run,
		})
		if err != nil {
			return nil, false, err
		}
		created = true
	}

	_, _, err = order.UpsertArtifact(tx, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeBranch,
		Key:  fmt.Sprintf("%s:branch:%s:%s", order.ID, strings.ToLower(repository), input.Head),
		Data: map[string]any{
			"name":       input.Head,
			"repository": repository,
		},
		Run: input.Run,
	})
	if err != nil {
		return nil, false, err
	}
	return pullRequest, created, nil
}

// taskPullRequestBody adds the closing keyword, the task link, and the
// SuperPlane footer to the description that the agent wrote.
func taskPullRequestBody(baseURL string, factory *models.Factory, order *models.FactoryWorkOrder, body string) string {
	sections := []string{}
	if origin := order.Origin(); origin != nil && origin.Label != "" {
		sections = append(sections, "Closes "+origin.Label)
	}
	path := order.URLPath(factory.RouteSegment())
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL != "" && path != "" {
		sections = append(sections, fmt.Sprintf("[%s](%s%s)", factory.WorkOrderKey(order.Number), baseURL, path))
	}
	if body = strings.TrimSpace(body); body != "" {
		sections = append(sections, body)
	}
	sections = append(sections, "<hr>", "Created via [SuperPlane](https://superplane.com).")
	return strings.Join(sections, "\n\n")
}
