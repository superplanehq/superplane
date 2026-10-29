package public

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

type suggestOwnersRequest struct {
	Paths []string `json:"paths"`
}

type assignOwnerRequest struct {
	User   string `json:"user"`
	Reason string `json:"reason"`
}

// taskOwnerTarget is the authorized task and the members who can own it.
type taskOwnerTarget struct {
	session    *models.FactoryPlanningSession
	repository string
	factory    *models.Factory
	order      *models.FactoryWorkOrder
	candidates []models.TaskOwnerCandidate
}

func (s *Server) handleRunnerPlanningSuggestOwners(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerSuggestOwners(w, r, s.planningOwnerTarget)
}

type ownerTargetLoader func(http.ResponseWriter, *http.Request) (*taskOwnerTarget, bool)

func (s *Server) planningOwnerTarget(w http.ResponseWriter, r *http.Request) (*taskOwnerTarget, bool) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return nil, false
	}
	target, err := s.loadPlanningTaskOwnerTarget(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, sessionOf(target), err)
		return nil, false
	}
	return target, true
}

func (s *Server) handleRunnerSuggestOwners(w http.ResponseWriter, r *http.Request, load ownerTargetLoader) {
	target, ok := load(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req suggestOwnersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	paths := taskowner.NormalizePaths(req.Paths)
	suggestion := taskowner.Suggestion{CodeOwners: []taskowner.CodeOwner{}, Members: taskowner.Members(target.candidates)}
	if len(paths) > 0 {
		suggestion = s.suggestCodeOwners(r.Context(), target, paths)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"paths":          paths,
		"current_owners": currentTaskOwners(target),
		"code_owners":    suggestion.CodeOwners,
		"members":        suggestion.Members,
		"history_note":   suggestion.HistoryNote,
	})
}

func (s *Server) suggestCodeOwners(ctx context.Context, target *taskOwnerTarget, paths []string) taskowner.Suggestion {
	return factoryactions.SuggestCodeOwners(ctx, s.ownerDeps(), target.factory, target.order, target.repository, paths, target.candidates)
}

func (s *Server) ownerDeps() factoryactions.IntakeDependencies {
	return factoryactions.IntakeDependencies{
		Registry:    s.registry,
		Encryptor:   s.encryptor,
		AuthService: s.authService,
	}
}

func currentTaskOwners(target *taskOwnerTarget) []taskowner.Member {
	owners := []taskowner.Member{}
	for _, assigneeID := range target.order.AssigneeIDs() {
		owner, ok := models.FindTaskOwnerCandidate(target.candidates, assigneeID.String())
		if !ok {
			owner = models.TaskOwnerCandidate{UserID: assigneeID}
		}
		owners = append(owners, taskowner.MemberOf(owner))
	}
	return owners
}

func (s *Server) handleRunnerPlanningAssignOwner(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerAssignOwner(w, r, s.planningOwnerTarget)
}

func (s *Server) handleRunnerAssignOwner(w http.ResponseWriter, r *http.Request, load ownerTargetLoader) {
	target, ok := load(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var req assignOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	reason := models.NormalizeTaskOwnerReason(req.Reason)
	if reason == "" {
		http.Error(w, "reason is required. Write one sentence that names the code.", http.StatusBadRequest)
		return
	}

	owner, ok := models.FindTaskOwnerCandidate(target.candidates, req.User)
	if !ok {
		http.Error(w, "The user cannot own tasks. Use a user_id or github_login from suggest_owners.", http.StatusBadRequest)
		return
	}

	err := assignTaskOwner(database.DB(r.Context()), target.order, models.TaskOwnerAssignment{
		UserID: owner.UserID,
		Source: models.TaskOwnerSourceAgent,
		Reason: reason,
	})
	if err != nil {
		if target.session != nil {
			writeRunnerPlanningError(w, r, target.session, err)
			return
		}
		log.WithError(err).Error("task owner assignment failed")
		http.Error(w, "Task owner assignment failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "assigned",
		"owner":  taskowner.MemberOf(owner),
	})
}

func assignTaskOwner(db *gorm.DB, order *models.FactoryWorkOrder, assignment models.TaskOwnerAssignment) error {
	changed, err := order.AssignOwner(db, assignment)
	if err != nil || !changed {
		return err
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		order.FactoryID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderAssigneesUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	return nil
}

// loadPlanningTaskOwnerTarget loads the task of a runner session. The runner
// token outlives the session, so an ended session or a closed task cannot
// change owners.
func (s *Server) loadPlanningTaskOwnerTarget(r *http.Request, scope *runneraction.PlanningSessionScope) (*taskOwnerTarget, error) {
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		return &taskOwnerTarget{session: session}, err
	}
	if session.State == models.PlanningSessionStateEnded {
		return &taskOwnerTarget{session: session}, models.ErrFactoryPlanningSessionEnded
	}
	target, err := s.loadPlanningTaskOwnerTargetForSession(r.Context(), session)
	if err != nil {
		return target, err
	}
	if target.order.State == models.FactoryWorkOrderStateClosed || target.order.IsArchived() {
		return target, models.ErrFactoryPlanningSessionEnded
	}
	return target, nil
}

func (s *Server) loadPlanningTaskOwnerTargetForSession(ctx context.Context, session *models.FactoryPlanningSession) (*taskOwnerTarget, error) {
	target := &taskOwnerTarget{session: session, repository: session.Repository}
	if session.DraftWorkOrderID == nil {
		return target, models.ErrFactoryWorkOrderNotFound
	}
	loaded, err := s.loadTaskOwnerTarget(ctx, session.OrganizationID, session.FactoryID, *session.DraftWorkOrderID, session.Repository)
	if loaded != nil {
		loaded.session = session
	}
	return loaded, err
}

func (s *Server) loadTaskOwnerTarget(ctx context.Context, orgID, factoryID, orderID uuid.UUID, repository string) (*taskOwnerTarget, error) {
	db := database.DB(ctx)
	target := &taskOwnerTarget{repository: repository}
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return target, err
	}
	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return target, err
	}
	if strings.TrimSpace(target.repository) == "" && order.Repository != nil {
		target.repository = *order.Repository
	}
	candidates, err := models.ListTaskOwnerCandidates(db, orgID, factoryID)
	if err != nil {
		return target, err
	}
	s.addTaskOwnerRoles(ctx, orgID, candidates)
	target.factory = factoryModel
	target.order = order
	target.candidates = candidates
	return target, nil
}

func (s *Server) addTaskOwnerRoles(ctx context.Context, orgID uuid.UUID, candidates []models.TaskOwnerCandidate) {
	factoryactions.AddTaskOwnerRoles(ctx, s.authService, orgID, candidates)
}

func (s *Server) taskOwnerHistory(ctx context.Context, factoryModel *models.Factory, repository string) (taskowner.History, error) {
	return factoryactions.TaskOwnerHistory(ctx, s.ownerDeps(), factoryModel, repository)
}

func sessionOf(target *taskOwnerTarget) *models.FactoryPlanningSession {
	if target == nil {
		return nil
	}
	return target.session
}
