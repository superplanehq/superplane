package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

var (
	ErrTaskPullRequestProviderUnsupported   = errors.New("pull requests open only in GitHub and Bitbucket workspaces")
	ErrTaskPullRequestGitHubNotConnected    = errors.New("the workspace GitHub integration is not connected")
	ErrTaskPullRequestBitbucketNotConnected = errors.New("the workspace Bitbucket integration is not connected")
	ErrTaskPullRequestOtherTask             = errors.New("the pull request belongs to another task")
	ErrTaskPullRequestGitHubTimeout         = errors.New("GitHub did not answer in time")
	ErrTaskPullRequestBitbucketTimeout      = errors.New("Bitbucket did not answer in time")
)

// taskPullRequestHostTimeout bounds the Git host calls of one open request,
// so the runner tool call returns before the agent stops waiting for it.
var taskPullRequestHostTimeout = 45 * time.Second

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

// taskRemotePullRequest is the Git host pull request that the task tracks.
type taskRemotePullRequest struct {
	Provider   string
	ExternalID string
	Repository string
	Number     int64
	URL        string
	Title      string
	Draft      bool
}

type taskPullRequestGitHubAPI interface {
	GetRepository(ctx context.Context, repository string) (*github.Repository, *github.Response, error)
	ListPullRequests(ctx context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	CreatePullRequest(ctx context.Context, repository string, pullRequest *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	EditPullRequest(ctx context.Context, repository string, pullNumber int, pullRequest *github.PullRequest) (*github.PullRequest, *github.Response, error)
}

var newTaskPullRequestGitHubAPI = func(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (taskPullRequestGitHubAPI, error) {
	return githubIntegrationClient(db, deps, factory)
}

type taskPullRequestBitbucketAPI interface {
	GetMainBranch(repository string) (string, error)
	ListPullRequests(repository string, opts bitbucketintegration.PullRequestListOptions) ([]map[string]any, error)
	CreatePullRequest(repository string, request bitbucketintegration.CreatePullRequestRequest) (map[string]any, error)
	UpdatePullRequest(repository string, id int64, request bitbucketintegration.UpdatePullRequestRequest) (map[string]any, error)
}

var newTaskPullRequestBitbucketAPI = func(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (taskPullRequestBitbucketAPI, error) {
	client, err := newFactoryBitbucketAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}
	return client, nil
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

	hostCtx, cancel := context.WithTimeout(ctx, taskPullRequestHostTimeout)
	defer cancel()
	var remote *taskRemotePullRequest
	if factory.OnboardingConfigValue().EffectiveVCSProvider() == models.ProviderBitbucket {
		input, remote, err = openBitbucketTaskPullRequest(hostCtx, db, deps, baseURL, factory, order, input)
	} else {
		input, remote, err = openGitHubTaskPullRequest(hostCtx, db, deps, baseURL, factory, order, input)
	}
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

func openGitHubTaskPullRequest(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	baseURL string,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (TaskPullRequestInput, *taskRemotePullRequest, error) {
	client, err := newTaskPullRequestGitHubAPI(db, deps, factory)
	if err != nil {
		return input, nil, ErrTaskPullRequestGitHubNotConnected
	}
	input, remote, err := openRemoteTaskPullRequest(ctx, db, client, baseURL, factory, order, input)
	if errors.Is(err, context.DeadlineExceeded) {
		return input, nil, fmt.Errorf("%w: %w", ErrTaskPullRequestGitHubTimeout, err)
	}
	if err != nil {
		return input, nil, err
	}
	return input, githubTaskRemotePullRequest(input.Repository, remote), nil
}

// openRemoteTaskPullRequest opens the pull request on GitHub, or updates the
// open one of the same branch, and returns the input with its base branch.
func openRemoteTaskPullRequest(
	ctx context.Context,
	db *gorm.DB,
	client taskPullRequestGitHubAPI,
	baseURL string,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (TaskPullRequestInput, *github.PullRequest, error) {
	if input.Base == "" {
		repository, _, err := client.GetRepository(ctx, input.Repository)
		if err != nil {
			return input, nil, err
		}
		input.Base = repository.GetDefaultBranch()
	}

	open, err := findOpenTaskPullRequest(ctx, client, input)
	if err != nil {
		return input, nil, err
	}
	if open != nil {
		if err := requireTaskOwnsPullRequest(db, factory, order, githubTaskRemotePullRequest(input.Repository, open)); err != nil {
			return input, nil, err
		}
	}
	body := taskPullRequestBody(baseURL, factory, order, input.Body, models.ProviderGitHub)
	remote, err := createOrUpdateTaskPullRequest(ctx, client, input, open, body)
	return input, remote, err
}

func openBitbucketTaskPullRequest(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	baseURL string,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (TaskPullRequestInput, *taskRemotePullRequest, error) {
	client, err := newTaskPullRequestBitbucketAPI(db, deps, factory)
	if err != nil {
		return input, nil, ErrTaskPullRequestBitbucketNotConnected
	}

	type result struct {
		input  TaskPullRequestInput
		remote *taskRemotePullRequest
		err    error
	}
	// shortcut: the Bitbucket client takes no context, so a late call keeps
	// running after the deadline. Pass the context when the client takes one.
	done := make(chan result, 1)
	go func() {
		input, remote, err := openRemoteBitbucketTaskPullRequest(db, client, baseURL, factory, order, input)
		done <- result{input: input, remote: remote, err: err}
	}()
	select {
	case outcome := <-done:
		return outcome.input, outcome.remote, outcome.err
	case <-ctx.Done():
		return input, nil, fmt.Errorf("%w: %w", ErrTaskPullRequestBitbucketTimeout, ctx.Err())
	}
}

// openRemoteBitbucketTaskPullRequest opens the pull request on Bitbucket, or
// updates the open one of the same branch, and returns the input with its
// base branch.
func openRemoteBitbucketTaskPullRequest(
	db *gorm.DB,
	client taskPullRequestBitbucketAPI,
	baseURL string,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	input TaskPullRequestInput,
) (TaskPullRequestInput, *taskRemotePullRequest, error) {
	if input.Base == "" {
		base, err := client.GetMainBranch(input.Repository)
		if err != nil {
			return input, nil, err
		}
		input.Base = base
	}

	open, err := client.ListPullRequests(input.Repository, bitbucketintegration.PullRequestListOptions{
		SourceBranch:      input.Head,
		DestinationBranch: input.Base,
		States:            []string{bitbucketintegration.PullRequestStateOpen},
	})
	if err != nil {
		return input, nil, err
	}

	body := taskPullRequestBody(baseURL, factory, order, input.Body, models.ProviderBitbucket)
	var raw map[string]any
	if len(open) > 0 {
		existing, err := bitbucketTaskRemotePullRequest(input.Repository, open[0])
		if err != nil {
			return input, nil, err
		}
		if err := requireTaskOwnsPullRequest(db, factory, order, existing); err != nil {
			return input, nil, err
		}
		raw, err = client.UpdatePullRequest(input.Repository, existing.Number, bitbucketintegration.UpdatePullRequestRequest{
			Title:       input.Title,
			Description: &body,
		})
		if err != nil {
			return input, nil, err
		}
	} else {
		raw, err = client.CreatePullRequest(input.Repository, bitbucketintegration.CreatePullRequestRequest{
			Title:       input.Title,
			Description: body,
			Source:      input.Head,
			Destination: input.Base,
			Draft:       input.Draft,
		})
		if err != nil {
			return input, nil, err
		}
	}
	remote, err := bitbucketTaskRemotePullRequest(input.Repository, raw)
	return input, remote, err
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
	switch factory.OnboardingConfigValue().EffectiveVCSProvider() {
	case models.ProviderGitHub, models.ProviderBitbucket:
	default:
		return input, ErrTaskPullRequestProviderUnsupported
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
// tracks. The check runs before the Git host edit, which cannot be undone.
func requireTaskOwnsPullRequest(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	open *taskRemotePullRequest,
) error {
	existing, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{
		Provider:   open.Provider,
		Repository: open.Repository,
		Number:     open.Number,
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

func githubTaskRemotePullRequest(repository string, remote *github.PullRequest) *taskRemotePullRequest {
	if fullName := remote.GetBase().GetRepo().GetFullName(); fullName != "" {
		repository = fullName
	}
	return &taskRemotePullRequest{
		Provider:   models.FactoryPullRequestProviderGitHub,
		ExternalID: strconv.FormatInt(remote.GetID(), 10),
		Repository: repository,
		Number:     int64(remote.GetNumber()),
		URL:        remote.GetHTMLURL(),
		Title:      remote.GetTitle(),
		Draft:      remote.GetDraft(),
	}
}

// bitbucketTaskRemotePullRequest reads a Bitbucket pull request response.
// Bitbucket pull requests have no global ID, so lookups use the repository
// and the number.
func bitbucketTaskRemotePullRequest(repository string, raw map[string]any) (*taskRemotePullRequest, error) {
	var remote struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Draft bool   `json:"draft"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		Destination struct {
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"destination"`
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &remote); err != nil {
		return nil, err
	}
	if remote.ID <= 0 {
		return nil, errors.New("bitbucket returned a pull request without an ID")
	}
	if fullName := remote.Destination.Repository.FullName; fullName != "" {
		repository = fullName
	}
	url := remote.Links.HTML.Href
	if url == "" {
		url = fmt.Sprintf("https://bitbucket.org/%s/pull-requests/%d", repository, remote.ID)
	}
	return &taskRemotePullRequest{
		Provider:   models.FactoryPullRequestProviderBitbucket,
		Repository: repository,
		Number:     remote.ID,
		URL:        url,
		Title:      remote.Title,
		Draft:      remote.Draft,
	}, nil
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
	remote *taskRemotePullRequest,
) (*models.FactoryPullRequest, bool, error) {
	state := models.FactoryPullRequestStateOpen
	if remote.Draft {
		state = models.FactoryPullRequestStateDraft
	}
	repository := remote.Repository
	title := remote.Title
	url := remote.URL

	existing, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{
		Provider:   remote.Provider,
		Repository: repository,
		Number:     remote.Number,
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
			Provider:   remote.Provider,
			ExternalID: remote.ExternalID,
			Repository: repository,
			Number:     remote.Number,
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

// taskPullRequestBody adds the task link and the SuperPlane footer to the
// description that the agent wrote. GitHub also gets the closing keyword.
// Bitbucket does not render HTML, so its footer uses a Markdown rule.
func taskPullRequestBody(baseURL string, factory *models.Factory, order *models.FactoryWorkOrder, body, provider string) string {
	sections := []string{}
	if origin := order.Origin(); origin != nil && origin.Label != "" && provider == models.ProviderGitHub {
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
	rule := "<hr>"
	if provider == models.ProviderBitbucket {
		rule = "---"
	}
	sections = append(sections, rule, "Created via [SuperPlane](https://superplane.com).")
	return strings.Join(sections, "\n\n")
}
