package public

import (
	"errors"
	"net/http"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
)

const runnerPullRequestBodyMaxBytes = 256 * 1024

type runnerOpenPullRequestRequest struct {
	Repository string `json:"repository"`
	Head       string `json:"head"`
	Base       string `json:"base"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
}

type runnerPullRequestView struct {
	Repository string `json:"repository"`
	Number     int64  `json:"number"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	State      string `json:"state"`
}

func (s *Server) handleRunnerOpenPullRequest(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopePullRequestsWrite)(w, r)
	if !ok {
		return
	}
	var request runnerOpenPullRequestRequest
	if err := decodeRunnerJSON(w, r, &request, runnerPullRequestBodyMaxBytes); err != nil {
		return
	}

	pullRequest, err := factoryactions.OpenTaskPullRequest(r.Context(), s.ownerDeps(), s.BaseURL, target.factory, target.order, factoryactions.TaskPullRequestInput{
		Repository: request.Repository,
		Head:       request.Head,
		Base:       request.Base,
		Title:      request.Title,
		Body:       request.Body,
		Draft:      request.Draft,
		Run:        &factoryevents.RunRef{ID: target.canvasRunID},
	})
	if err != nil {
		writeRunnerPullRequestError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, runnerPullRequestViewFrom(pullRequest))
}

func (s *Server) handleRunnerListPullRequests(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopePullRequestsRead)(w, r)
	if !ok {
		return
	}
	pullRequests, err := factoryactions.ListTaskPullRequests(database.DB(r.Context()), target.order)
	if err != nil {
		writeRunnerPullRequestError(w, err)
		return
	}
	views := make([]runnerPullRequestView, 0, len(pullRequests))
	for i := range pullRequests {
		views = append(views, runnerPullRequestViewFrom(&pullRequests[i]))
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"pull_requests": views})
}

func runnerPullRequestViewFrom(pullRequest *models.FactoryPullRequest) runnerPullRequestView {
	return runnerPullRequestView{
		Repository: pullRequest.Repository,
		Number:     pullRequest.Number,
		URL:        pullRequest.URL,
		Title:      pullRequest.Title,
		State:      pullRequest.State,
	}
}

func writeRunnerPullRequestError(w http.ResponseWriter, err error) {
	var inputErr *factoryactions.TaskPullRequestInputError
	var githubErr *github.ErrorResponse
	switch {
	case errors.As(err, &inputErr):
		http.Error(w, inputErr.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryPullRequestRepositoryNotInWorkspace):
		http.Error(w, "The pull request repository is not in this workspace.", http.StatusBadRequest)
	case errors.Is(err, factoryactions.ErrTaskPullRequestGitHubOnly),
		errors.Is(err, factoryactions.ErrTaskPullRequestGitHubNotConnected),
		errors.Is(err, factoryactions.ErrTaskPullRequestOtherTask),
		errors.Is(err, models.ErrFactoryPullRequestAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, factoryactions.ErrTaskPullRequestGitHubTimeout):
		http.Error(w, "GitHub did not answer in time. Try again later.", http.StatusGatewayTimeout)
	case errors.As(err, &githubErr):
		http.Error(w, "GitHub rejected the pull request: "+githubErr.Message, http.StatusBadGateway)
	default:
		log.WithError(err).Error("runner pull request request failed")
		http.Error(w, "Pull request request failed", http.StatusInternalServerError)
	}
}
