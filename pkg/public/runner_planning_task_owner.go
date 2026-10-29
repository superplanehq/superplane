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

const taskOwnerHistoryUnavailable = "SuperPlane could not read the code history. Use the members list."

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
	history, err := s.taskOwnerHistory(ctx, target.factory, target.repository)
	suggestion := taskowner.Suggestion{CodeOwners: []taskowner.CodeOwner{}, Members: taskowner.Members(target.candidates)}
	if err == nil {
		suggestion, err = taskowner.Suggest(ctx, history, paths, target.candidates, time.Now())
	}
	if err != nil {
		log.WithError(err).Warnf("task owner history unavailable for order %s", target.order.ID)
		suggestion.HistoryNote = taskOwnerHistoryUnavailable
	}
	return suggestion
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

// addTaskOwnerRoles adds the organization roles that the Members page shows.
// The agent can apply role rules from the instructions. A lookup error leaves
// the roles empty: the members stay usable.
func (s *Server) addTaskOwnerRoles(ctx context.Context, orgID uuid.UUID, candidates []models.TaskOwnerCandidate) {
	rolesByUser, err := s.organizationRolesByUser(ctx, orgID.String())
	if err != nil {
		log.WithError(err).Warnf("task owner roles unavailable for organization %s", orgID)
	}
	for i := range candidates {
		roles := []string{}
		if candidates[i].IsOwner {
			roles = append(roles, models.DisplayNameOwner)
		}
		candidates[i].Roles = append(roles, rolesByUser[candidates[i].UserID.String()]...)
	}
}

func (s *Server) organizationRolesByUser(ctx context.Context, orgID string) (map[string][]string, error) {
	if s.authService == nil {
		return nil, errors.New("authorization service is unavailable")
	}
	definitions, err := s.authService.GetAllRoleDefinitions(ctx, models.DomainTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}
	metadata, err := models.FindRoleMetadataByNames(names, models.DomainTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}

	rolesByUser := map[string][]string{}
	for _, definition := range definitions {
		userIDs, err := s.authService.GetOrgUsersForRole(ctx, definition.Name, orgID)
		if err != nil {
			return nil, err
		}
		displayName := definition.Name
		if roleMetadata, ok := metadata[definition.Name]; ok && roleMetadata.DisplayName != "" {
			displayName = roleMetadata.DisplayName
		}
		for _, userID := range userIDs {
			rolesByUser[userID] = append(rolesByUser[userID], displayName)
		}
	}
	return rolesByUser, nil
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

func sessionOf(target *taskOwnerTarget) *models.FactoryPlanningSession {
	if target == nil {
		return nil
	}
	return target.session
}
