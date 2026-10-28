package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	taskOwnerHistoryUnavailable = "SuperPlane could not read the code history. Pick from members by open tasks."
	taskOwnerFallbackReason     = "No owner was chosen during planning. SuperPlane picked the member with the fewest open tasks."
)

type planningSuggestOwnersRequest struct {
	Paths []string `json:"paths"`
}

type planningAssignOwnerRequest struct {
	User   string `json:"user"`
	Reason string `json:"reason"`
}

type planningTaskOwnerRef struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	GitHubLogin string    `json:"github_login"`
}

// planningTaskOwnerTarget is the session task and the members who can own it.
type planningTaskOwnerTarget struct {
	session    *models.FactoryPlanningSession
	factory    *models.Factory
	order      *models.FactoryWorkOrder
	candidates []models.TaskOwnerCandidate
}

func (s *Server) handleRunnerPlanningSuggestOwners(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningSuggestOwnersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	target, err := s.loadPlanningTaskOwnerTarget(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, sessionOf(target), err)
		return
	}

	paths := taskowner.NormalizePaths(req.Paths)
	if len(target.order.Assignees) > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"task_has_owner": true, "paths": paths})
		return
	}
	history, err := s.taskOwnerHistory(r.Context(), target.factory, target.session.Repository)
	suggestion := taskowner.Suggestion{CodeOwners: []taskowner.CodeOwner{}, Members: taskowner.Members(target.candidates)}
	if err == nil {
		suggestion, err = taskowner.Suggest(r.Context(), history, paths, target.candidates, time.Now())
	}
	if err != nil {
		log.WithError(err).Warnf("task owner history unavailable for order %s", target.order.ID)
		suggestion.HistoryNote = taskOwnerHistoryUnavailable
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task_has_owner": false,
		"paths":          paths,
		"code_owners":    suggestion.CodeOwners,
		"members":        suggestion.Members,
		"history_note":   suggestion.HistoryNote,
	})
}

func (s *Server) handleRunnerPlanningAssignOwner(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningAssignOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	reason := models.NormalizeTaskOwnerReason(req.Reason)
	if reason == "" {
		http.Error(w, "reason is required. Write one sentence that names the code.", http.StatusBadRequest)
		return
	}
	target, err := s.loadPlanningTaskOwnerTarget(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, sessionOf(target), err)
		return
	}

	owner, ok := models.FindTaskOwnerCandidate(target.candidates, req.User)
	if !ok {
		http.Error(w, "The user cannot own tasks. Use a user_id or github_login from suggest_owners.", http.StatusBadRequest)
		return
	}

	assigned, err := assignPlanningTaskOwner(database.DB(r.Context()), target.order, models.TaskOwnerAssignment{
		UserID: owner.UserID,
		Source: models.TaskOwnerSourceAgent,
		Reason: reason,
	})
	if err != nil {
		writeRunnerPlanningError(w, r, target.session, err)
		return
	}
	if !assigned {
		writeJSON(w, http.StatusOK, map[string]any{"status": "already_owned"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "assigned",
		"owner":  planningTaskOwnerRef{UserID: owner.UserID, Name: owner.Name, GitHubLogin: owner.GitHubLogin},
	})
}

// assignFallbackTaskOwner makes sure planning never leaves a task without an
// owner. It assigns the member with the fewest open tasks when the agent did
// not pick one. Errors are logged: the plan update already succeeded.
func (s *Server) assignFallbackTaskOwner(r *http.Request, session *models.FactoryPlanningSession) {
	target, err := s.loadPlanningTaskOwnerTargetForSession(database.DB(r.Context()), session)
	if err != nil {
		log.WithError(err).Warnf("fallback task owner skipped for session %s", session.ID)
		return
	}
	if len(target.order.Assignees) > 0 {
		return
	}
	owner, ok := models.LeastLoadedTaskOwner(target.candidates)
	if !ok {
		return
	}
	_, err = assignPlanningTaskOwner(database.DB(r.Context()), target.order, models.TaskOwnerAssignment{
		UserID: owner.UserID,
		Source: models.TaskOwnerSourceFallback,
		Reason: taskOwnerFallbackReason,
	})
	if err != nil {
		log.WithError(err).Warnf("fallback task owner failed for order %s", target.order.ID)
	}
}

// planningSessionHasScores reports whether planning already scored the task.
// The first scored update ends the first planning turn, which is when the
// task must have an owner.
func planningSessionHasScores(db *gorm.DB, session *models.FactoryPlanningSession) bool {
	if session.DraftWorkOrderID == nil {
		return true
	}
	checks, err := models.ListChecksForWorkOrders(db, []uuid.UUID{*session.DraftWorkOrderID})
	if err != nil {
		log.WithError(err).Warnf("fallback task owner skipped for session %s", session.ID)
		return true
	}
	return len(checks[*session.DraftWorkOrderID]) > 0
}

func assignPlanningTaskOwner(db *gorm.DB, order *models.FactoryWorkOrder, assignment models.TaskOwnerAssignment) (bool, error) {
	assigned, err := order.AssignOwnerIfUnowned(db, assignment)
	if err != nil || !assigned {
		return assigned, err
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		order.FactoryID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderAssigneesUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	messages.PublishWorkOrderOwnersAssigned(order, nil, []uuid.UUID{assignment.UserID}, models.NormalizeTaskOwnerReason(assignment.Reason))
	return true, nil
}

// loadPlanningTaskOwnerTarget loads the task of a runner session. The runner
// token outlives the session, so an ended session or a closed task cannot
// change owners.
func (s *Server) loadPlanningTaskOwnerTarget(r *http.Request, scope *runneraction.PlanningSessionScope) (*planningTaskOwnerTarget, error) {
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		return &planningTaskOwnerTarget{session: session}, err
	}
	if session.State == models.PlanningSessionStateEnded {
		return &planningTaskOwnerTarget{session: session}, models.ErrFactoryPlanningSessionEnded
	}
	target, err := s.loadPlanningTaskOwnerTargetForSession(database.DB(r.Context()), session)
	if err != nil {
		return target, err
	}
	if target.order.State == models.FactoryWorkOrderStateClosed || target.order.IsArchived() {
		return target, models.ErrFactoryPlanningSessionEnded
	}
	return target, nil
}

func (s *Server) loadPlanningTaskOwnerTargetForSession(db *gorm.DB, session *models.FactoryPlanningSession) (*planningTaskOwnerTarget, error) {
	target := &planningTaskOwnerTarget{session: session}
	if session.DraftWorkOrderID == nil {
		return target, models.ErrFactoryWorkOrderNotFound
	}
	factoryModel, err := models.FindFactory(db, session.OrganizationID, session.FactoryID)
	if err != nil {
		return target, err
	}
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	if err != nil {
		return target, err
	}
	candidates, err := models.ListTaskOwnerCandidates(db, session.OrganizationID, session.FactoryID)
	if err != nil {
		return target, err
	}
	target.factory = factoryModel
	target.order = order
	target.candidates = candidates
	return target, nil
}

// taskOwnerHistory reads code history through the GitHub integration of the
// factory. Bitbucket factories have no history source yet.
func (s *Server) taskOwnerHistory(ctx context.Context, factoryModel *models.Factory, repository string) (taskowner.History, error) {
	config := factoryModel.OnboardingConfigValue()
	if !strings.EqualFold(effectiveTaskOwnerProvider(config.VCSProvider), models.ProviderGitHub) {
		return nil, fmt.Errorf("provider %q has no code history source", config.VCSProvider)
	}
	repository = strings.TrimSpace(repository)
	if repository == "" {
		repository = strings.TrimSpace(config.AppRepository)
	}
	if repository == "" {
		return nil, errors.New("factory has no repository")
	}
	integrationID, err := uuid.Parse(strings.TrimSpace(config.VCSIntegrationID))
	if err != nil {
		return nil, errors.New("factory has no GitHub integration")
	}
	if s.registry == nil {
		return nil, errors.New("integration registry is unavailable")
	}

	instance, err := models.FindIntegration(factoryModel.OrganizationID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("load integration: %w", err)
	}
	if instance.AppName != "github" || instance.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not a ready GitHub integration", integrationID)
	}
	integrationCtx := contexts.NewIntegrationContext(database.DB(ctx), nil, instance, s.encryptor, s.registry, nil)
	client, err := common.NewClient(integrationCtx, s.registry.HTTPContext())
	if err != nil {
		return nil, fmt.Errorf("build GitHub client: %w", err)
	}
	return taskowner.GitHubHistory{Client: client, Repository: repository}, nil
}

func effectiveTaskOwnerProvider(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return models.ProviderGitHub
	}
	return provider
}

func sessionOf(target *planningTaskOwnerTarget) *models.FactoryPlanningSession {
	if target == nil {
		return nil
	}
	return target.session
}
