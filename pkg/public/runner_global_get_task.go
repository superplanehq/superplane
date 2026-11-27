package public

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type runnerTaskDetailView struct {
	runnerTaskView
	Description  string                        `json:"description"`
	Column       string                        `json:"column,omitempty"`
	PullRequests []runnerTaskPullRequestDetail `json:"pull_requests"`
}

type runnerTaskPullRequestDetail struct {
	runnerPullRequestView
	HeadSHA string `json:"head_sha,omitempty"`
}

// handleRunnerGetTask finds one task from a reference in a webhook payload,
// such as a task key, a pull request, or a commit.
func (s *Server) handleRunnerGetTask(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeTasksRead)
	if !ok {
		return
	}
	query := r.URL.Query()
	task := strings.TrimSpace(query.Get("task"))
	pullRequest := strings.TrimSpace(query.Get("pull_request"))
	repository := strings.TrimSpace(query.Get("repository"))
	number := strings.TrimSpace(query.Get("number"))
	commit := strings.TrimSpace(query.Get("commit"))
	selectors := slices.DeleteFunc([]string{task, pullRequest, number, commit}, func(value string) bool { return value == "" })
	if len(selectors) != 1 || (repository != "" && number == "") {
		http.Error(w, "Pass exactly one of task, pull_request, number with an optional repository, or commit", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	var order *models.FactoryWorkOrder
	switch {
	case task != "":
		order, err = factoryModel.FindWorkOrderByRef(db, task)
	case pullRequest != "":
		var found *models.FactoryPullRequest
		found, err = factoryModel.FindPullRequest(db, models.FactoryPullRequestLookup{URL: pullRequest})
		if err == nil {
			order, err = runnerPullRequestTask(db, factoryModel, []models.FactoryPullRequest{*found})
		}
	case number != "":
		parsed, parseErr := strconv.ParseInt(number, 10, 64)
		if parseErr != nil || parsed < 1 {
			http.Error(w, "number must be a positive integer", http.StatusBadRequest)
			return
		}
		var found []models.FactoryPullRequest
		found, err = factoryModel.ListPullRequestsByNumber(db, repository, parsed)
		if err == nil {
			order, err = runnerPullRequestTask(db, factoryModel, found)
		}
	default:
		var found []models.FactoryPullRequest
		found, err = factoryModel.ListPullRequestsByCommitSHA(db, commit)
		if err == nil {
			order, err = runnerPullRequestTask(db, factoryModel, found)
		}
	}
	var ambiguous *runnerAmbiguousTaskError
	if errors.As(err, &ambiguous) {
		http.Error(w, "Several tasks match: "+strings.Join(ambiguous.keys, ", ")+". Pass repository or a more exact selector", http.StatusConflict)
		return
	}
	if err == nil && order.IsArchived() {
		err = models.ErrFactoryWorkOrderNotFound
	}
	switch {
	case errors.Is(err, models.ErrFactoryWorkOrderNotFound), errors.Is(err, models.ErrFactoryPullRequestNotFound):
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	case errors.Is(err, models.ErrFactoryPullRequestInvalid):
		http.Error(w, "pull_request must be a pull request URL", http.StatusBadRequest)
		return
	case err != nil:
		writeRunnerTaskError(w, err)
		return
	}

	view, err := s.runnerTaskDetail(db, factoryModel, order)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, view)
}

type runnerAmbiguousTaskError struct {
	keys []string
}

func (e *runnerAmbiguousTaskError) Error() string {
	return "several tasks match: " + strings.Join(e.keys, ", ")
}

// runnerPullRequestTask returns the one task of the pull requests. More than
// one task is ambiguous, so the error lists their keys.
func runnerPullRequestTask(tx *gorm.DB, factoryModel *models.Factory, pullRequests []models.FactoryPullRequest) (*models.FactoryWorkOrder, error) {
	var ids []uuid.UUID
	for _, pullRequest := range pullRequests {
		if !slices.Contains(ids, pullRequest.WorkOrderID) {
			ids = append(ids, pullRequest.WorkOrderID)
		}
	}
	if len(ids) == 0 {
		return nil, models.ErrFactoryWorkOrderNotFound
	}
	if len(ids) == 1 {
		return factoryModel.FindWorkOrder(tx, ids[0])
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		order, err := factoryModel.FindWorkOrder(tx, id)
		if err != nil {
			return nil, err
		}
		keys = append(keys, factoryModel.WorkOrderKey(order.Number))
	}
	return nil, &runnerAmbiguousTaskError{keys: keys}
}

func (s *Server) runnerTaskDetail(tx *gorm.DB, factoryModel *models.Factory, order *models.FactoryWorkOrder) (runnerTaskDetailView, error) {
	view := runnerTaskDetailView{runnerTaskView: runnerTaskViewFrom(s.BaseURL, factoryModel, order), Description: order.Description}
	column, err := models.WorkOrderBoardColumn(tx, order)
	if err != nil {
		return view, err
	}
	if order.IsOpen() {
		dispatch, err := order.LatestLineDispatch(tx)
		if err != nil && !errors.Is(err, models.ErrFactoryWorkOrderLineDispatchNotFound) {
			return view, err
		}
		if dispatch != nil && dispatch.CurrentStageKey != "" {
			column = dispatch.CurrentStageKey
		}
	}
	view.Column = column
	pullRequests, err := factoryactions.ListTaskPullRequests(tx, order)
	if err != nil {
		return view, err
	}
	view.PullRequests = make([]runnerTaskPullRequestDetail, 0, len(pullRequests))
	for i := range pullRequests {
		detail := runnerTaskPullRequestDetail{runnerPullRequestView: runnerPullRequestViewFrom(&pullRequests[i])}
		if pullRequests[i].CurrentRevisionID != nil {
			revision, err := models.FindPullRequestRevision(tx, *pullRequests[i].CurrentRevisionID)
			if err != nil {
				return view, err
			}
			detail.HeadSHA = revision.SHA
		}
		view.PullRequests = append(view.PullRequests, detail)
	}
	return view, nil
}
