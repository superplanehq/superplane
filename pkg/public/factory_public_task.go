package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// This is an allowlist, not the private work-order serializer. Never add run
// payloads, artifacts, signed file URLs, or usage costs to this response.
type publicTask struct {
	Card         publicCard          `json:"card"`
	Description  string              `json:"description"`
	Column       string              `json:"column"`
	TotalTokens  int64               `json:"totalTokens"`
	Comments     []publicTaskComment `json:"comments"`
	Runs         []publicTaskRun     `json:"runs"`
	PullRequests []publicTaskPR      `json:"pullRequests"`
}
type publicTaskComment struct {
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}
type publicTaskRun struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	Result          string `json:"result,omitempty"`
	DurationSeconds int64  `json:"durationSeconds"`
}
type publicTaskPR struct {
	Number int64  `json:"number"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

func (s *Server) handlePublicFactoryTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	task, err := loadPublicFactoryTask(r.Context(), mux.Vars(r))
	if err != nil {
		if errors.Is(err, errPublicBoardNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		log.WithError(err).Error("failed to load public task")
		http.Error(w, "Failed to load task", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		log.WithError(err).Error("failed to write public task")
	}
}

func loadPublicFactoryTask(ctx context.Context, vars map[string]string) (*publicTask, error) {
	number, err := strconv.ParseInt(vars["number"], 10, 64)
	if err != nil || number <= 0 {
		return nil, errPublicBoardNotFound
	}
	factory, line, err := authorizePublicFactoryLine(ctx, vars)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	// Use the same projection as the board. Guessing a number must not expose
	// archived tasks, another workspace, or tasks absent from this line.
	board, err := buildPublicBoard(db, factory, line)
	if err != nil {
		return nil, err
	}
	task := &publicTask{Comments: []publicTaskComment{}, Runs: []publicTaskRun{}, PullRequests: []publicTaskPR{}}
	found := false
	for _, column := range board.Columns {
		for _, card := range column.Cards {
			if card.Number == number {
				task.Card = card
				task.Column = column.Title
				found = true
			}
		}
	}
	if !found {
		return nil, errPublicBoardNotFound
	}
	order, err := factory.FindWorkOrderByNumber(db, number)
	if err != nil {
		return nil, err
	}
	if order.IsArchived() {
		return nil, errPublicBoardNotFound
	}
	task.Description = order.Description
	comments, err := order.ListComments(db)
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		task.Comments = append(task.Comments, publicTaskComment{Body: comment.Body, CreatedAt: comment.CreatedAt})
	}
	ids := []uuid.UUID{order.ID}
	usage, err := models.SumUsageForWorkOrders(db, ids)
	if err != nil {
		return nil, err
	}
	task.TotalTokens = usage[order.ID].TotalTokens
	dispatches, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, ids)
	if err != nil {
		return nil, err
	}
	for _, dispatch := range dispatches[order.ID] {
		if dispatch.LineID != line.ID {
			continue
		}
		for _, execution := range dispatch.Executions {
			task.Runs = append(task.Runs, publicTaskRun{Name: execution.StepName, Status: execution.Status, Result: executionResult(&execution), DurationSeconds: execution.DurationSeconds})
		}
	}
	prs, err := models.ListPullRequestsByWorkOrderIDs(db, ids)
	if err != nil {
		return nil, err
	}
	for _, pr := range prs[order.ID] {
		task.PullRequests = append(task.PullRequests, publicTaskPR{Number: pr.Number, State: pr.State, URL: publicTaskLink(pr.URL)})
	}
	// Recheck visibility after loading details in case the owner revoked access.
	if _, _, err := authorizePublicFactoryLine(ctx, vars); err != nil {
		return nil, err
	}
	return task, nil
}

func publicTaskLink(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
