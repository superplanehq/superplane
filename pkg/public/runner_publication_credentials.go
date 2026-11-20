package public

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	githubintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var publicationRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type publicationHTTPContext struct {
	core.HTTPContext
	ctx context.Context
}

func (h publicationHTTPContext) Do(request *http.Request) (*http.Response, error) {
	return h.HTTPContext.Do(request.WithContext(h.ctx))
}

// This internal endpoint is not registered as an agent-visible credential tool.
func (s *Server) handleRunnerPublicationCredential(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopePullRequestsWrite)
	if !ok {
		return
	}
	db := database.DB(r.Context())
	execution, err := models.FindNodeExecutionInTransaction(db, scope.CanvasID, scope.NodeExecutionID)
	if scope.Flow != runneraction.RunnerFlowTask || err != nil || execution.State != models.CanvasNodeExecutionStateStarted {
		http.Error(w, "Publication requires an active implementation execution", http.StatusForbidden)
		return
	}
	implementation, err := models.FindWorkOrderExecutionForRun(db, scope.CanvasRunID)
	if err != nil || implementation.Status != models.FactoryWorkOrderExecutionStatusRunning || implementation.WorkOrderID != scope.WorkOrderID || implementation.FactoryID != scope.FactoryID || implementation.OrganizationID != scope.OrganizationID {
		http.Error(w, "Implementation execution is inactive", http.StatusForbidden)
		return
	}
	target, ok := s.runnerTaskTarget(w, r, scope)
	if !ok {
		return
	}
	var request struct {
		Repository string `json:"repository"`
		Host       string `json:"host"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || !publicationRepositoryPattern.MatchString(request.Repository) {
		http.Error(w, "Invalid publication credential request", http.StatusBadRequest)
		return
	}
	allowed, err := target.factory.IsTaskRepository(db, target.order, request.Repository)
	if err != nil || !allowed || target.factory.OnboardingConfigValue().EffectiveVCSProvider() != models.ProviderGitHub {
		http.Error(w, "Repository is not authorized for publication", http.StatusForbidden)
		return
	}
	id, err := uuid.Parse(target.factory.OnboardingConfigValue().VCSIntegrationID)
	if err != nil {
		http.Error(w, "GitHub is not connected", http.StatusConflict)
		return
	}
	integration, err := models.FindIntegrationInTransaction(db, scope.OrganizationID, id)
	if err != nil || integration.State != models.IntegrationStateReady || integration.AppName != models.ProviderGitHub {
		http.Error(w, "GitHub is not connected", http.StatusConflict)
		return
	}
	host := common.HostFromModel(integration)
	if !strings.EqualFold(request.Host, host.GitHost()) {
		http.Error(w, "Publication host does not match the GitHub connection", http.StatusForbidden)
		return
	}
	integrationCtx := contexts.NewIntegrationContext(db, nil, integration, s.encryptor, s.registry, nil)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	token, err := githubintegration.ResolvePublicationToken(publicationHTTPContext{s.registry.HTTPContextInTransaction(db), ctx}, integrationCtx, request.Repository)
	if err != nil || token == "" {
		// Provider errors can contain credentials. Do not log or return them.
		http.Error(w, "GitHub publication credential resolution failed. Check the connection and repository grants.", http.StatusBadGateway)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]string{"token": token, "host": host.GitHost()})
}
