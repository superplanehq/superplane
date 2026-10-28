package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	appcatalog "github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

const (
	githubInstallApprovedPath         = "/github/approved"
	githubInstallRequestRefreshWindow = 2 * time.Minute
)

var requestGitHubAppInstallRequestRefresh = func(ctx context.Context, until time.Time) error {
	return models.RequestVCSProviderInstallRequestRefresh(database.DB(ctx), models.ProviderGitHub, until)
}

var enqueueGitHubAppInstallationReconciliation = func(
	ctx context.Context,
	installationID int64,
	organizationID uuid.UUID,
	availableAt time.Time,
) error {
	return models.EnqueueVCSProviderInstallationReconciliation(
		database.DB(ctx),
		models.ProviderGitHub,
		installationID,
		organizationID,
		availableAt,
	)
}

var hasGitHubAppInstallationRequest = func(ctx context.Context, installationID int64) (bool, error) {
	db := database.DB(ctx)
	installation, err := models.FindVCSProviderInstallation(db, models.ProviderGitHub, installationID)
	if err == nil {
		return models.HasVCSProviderInstallRequestForAccount(
			db,
			models.ProviderGitHub,
			installation.AccountID,
			installation.AccountLogin,
		)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("find GitHub App installation %d: %w", installationID, err)
	}

	cfg, resolveErr := appcatalog.ResolveProcess(ctx)
	if resolveErr != nil {
		return false, resolveErr
	}
	catalog, err := appcatalog.NewCatalog(db, cfg)
	if err != nil {
		return false, err
	}
	return catalog.HasInstallationRequest(ctx, installationID)
}

// HandleGitHubAppSetup handles only GitHub's installation and repository
// settings redirect. Signed webhooks are authoritative for catalog state.
func (s *Server) HandleGitHubAppSetup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("setup_action") == "request" {
		s.handleGitHubAppInstallRequest(w, r)
		return
	}
	installationID, err := strconv.ParseInt(strings.TrimSpace(query.Get("installation_id")), 10, 64)
	if err != nil || installationID <= 0 {
		http.Error(w, "missing installation id", http.StatusBadRequest)
		return
	}
	organizationID := githubAppSetupOrganizationIDAt(r.Context(), query.Get("state"))

	switch query.Get("setup_action") {
	case "install", "update":
		redirectPath := s.githubAppSetupRedirect(r, organizationID, query.Get("state"))
		if query.Get("setup_action") == "install" && strings.TrimSpace(query.Get("state")) == "" {
			requested, requestErr := hasGitHubAppInstallationRequest(r.Context(), installationID)
			if requestErr != nil {
				log.WithError(requestErr).WithField("installation_id", installationID).Warn(
					"failed to identify GitHub App installation request",
				)
			} else if requested {
				redirectPath = githubInstallApprovedPath
			}
		}
		if err := enqueueGitHubAppInstallationReconciliation(r.Context(), installationID, organizationID, time.Now()); err != nil {
			http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, redirectPath, http.StatusFound)
	default:
		http.Error(w, "invalid setup action", http.StatusBadRequest)
	}
}

// githubAppSetupRedirect opens Choose organization for one unfinished workspace.
// A signed install state names the organization, not the workspace. Without
// state, the signed-in session can name a workspace only when that session has
// one unfinished workspace. Two unfinished workspaces return home so install
// does not open the wrong one. This does not create a workspace.
func (s *Server) githubAppSetupRedirect(r *http.Request, organizationID uuid.UUID, state string) string {
	if strings.TrimSpace(state) != "" {
		path, err := incompleteOrganizationSetupPath(r.Context(), organizationID)
		if err != nil {
			log.WithError(err).WithField("organization_id", organizationID).Warn(
				"failed to resolve GitHub App setup return",
			)
			return "/"
		}
		return path
	}

	account, ok := middleware.AccountFromRequest(r, s.jwt)
	if !ok {
		return "/"
	}
	path, err := singleIncompleteSessionSetupPath(r.Context(), account)
	if err != nil {
		log.WithError(err).WithField("account_id", account.ID).Warn(
			"failed to resolve GitHub App setup return",
		)
		return "/"
	}
	return path
}

func incompleteOrganizationSetupPath(ctx context.Context, organizationID uuid.UUID) (string, error) {
	if organizationID == uuid.Nil {
		return "/", nil
	}
	db := database.DB(ctx)
	factories, err := models.ListFactories(db, organizationID)
	if err != nil {
		return "", err
	}
	factory := onlyIncompleteFactory(factories)
	if factory == nil {
		return "/", nil
	}
	organization, err := models.FindOrganizationByIDInTransaction(db, organizationID.String())
	if err != nil {
		return "", err
	}
	return workspaceSetupRepoPath(organization.Slug, factory.RouteSegment()), nil
}

func singleIncompleteSessionSetupPath(ctx context.Context, account *models.Account) (string, error) {
	if account == nil {
		return "/", nil
	}
	organizations, err := models.FindOrganizationsForAccount(account.Email)
	if err != nil {
		return "", err
	}
	db := database.DB(ctx)
	var path string
	for _, organization := range organizations {
		factories, err := models.ListFactories(db, organization.ID)
		if err != nil {
			return "", err
		}
		for _, factory := range factories {
			if factory.IsOnboardingComplete() {
				continue
			}
			if path != "" {
				return "/", nil
			}
			path = workspaceSetupRepoPath(organization.Slug, factory.RouteSegment())
		}
	}
	if path == "" {
		return "/", nil
	}
	return path, nil
}

func onlyIncompleteFactory(factories []models.Factory) *models.Factory {
	var incomplete *models.Factory
	for index := range factories {
		if factories[index].IsOnboardingComplete() {
			continue
		}
		if incomplete != nil {
			return nil
		}
		incomplete = &factories[index]
	}
	return incomplete
}

func workspaceSetupRepoPath(slug, segment string) string {
	slug = strings.TrimSpace(slug)
	segment = strings.TrimSpace(segment)
	if slug == "" || segment == "" || strings.ContainsAny(slug, "/?#") || strings.ContainsAny(segment, "/?#") {
		return "/"
	}
	target := url.URL{Path: "/" + slug + "/workspaces/" + segment + "/setup"}
	query := url.Values{}
	query.Set("step", "repo")
	target.RawQuery = query.Encode()
	return target.String()
}

// Anyone can open this redirect, so it must not call GitHub. It only asks the
// catalog worker to read the request list often for a short time.
func (s *Server) handleGitHubAppInstallRequest(w http.ResponseWriter, r *http.Request) {
	until := time.Now().Add(githubInstallRequestRefreshWindow)
	if err := requestGitHubAppInstallRequestRefresh(r.Context(), until); err != nil {
		http.Error(w, "failed to queue GitHub App request refresh", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func githubAppSetupOrganizationID(state string) uuid.UUID {
	return githubAppSetupOrganizationIDAt(context.Background(), state)
}

func githubAppSetupOrganizationIDAt(ctx context.Context, state string) uuid.UUID {
	cfg, err := appcatalog.ResolveProcess(ctx)
	if err != nil || !cfg.Enabled() {
		return uuid.Nil
	}
	organizationID, err := common.VerifyHostedAppInstallState(cfg.WebhookSecret, state)
	if err != nil {
		return uuid.Nil
	}
	return organizationID
}

// HandleGitHubAppWebhook validates the public App signature and updates the
// global catalog. Repository events reach nodes through the repository hook
// that WebhookProvisioner registers for each webhook, so this endpoint does
// not deliver them.
func (s *Server) HandleGitHubAppWebhook(w http.ResponseWriter, r *http.Request) {
	app, ok := common.ResolveHostedApp(r.Context())
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	payload, err := gh.ValidatePayload(r, []byte(app.WebhookSecret))
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	event, err := gh.ParseWebHook(gh.WebHookType(r), payload)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	installationID, ok := githubInstallationID(payload)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := applyGitHubCatalogWebhook(database.DB(r.Context()), event, installationID); err != nil {
		log.WithError(err).Error("failed to update the GitHub App catalog")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func applyGitHubCatalogWebhook(tx *gorm.DB, event any, installationID int64) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		return applyGitHubCatalogWebhookInTransaction(tx, event, installationID)
	})
}

func applyGitHubCatalogWebhookInTransaction(tx *gorm.DB, event any, installationID int64) error {
	switch event := event.(type) {
	case *gh.InstallationEvent:
		if event.GetAction() == "deleted" {
			return models.DeleteVCSProviderInstallation(tx, models.ProviderGitHub, installationID)
		}
		if event.GetInstallation() == nil {
			return nil
		}
		installation := githubInstallationModel(event.GetInstallation())
		if err := models.UpsertVCSProviderInstallation(tx, &installation); err != nil {
			return err
		}
		if event.GetAction() != "created" {
			return models.DeleteVCSProviderInstallRequestsForAccount(
				tx,
				models.ProviderGitHub,
				installation.AccountID,
				installation.AccountLogin,
			)
		}
		repositories := githubRepositoryModels(installationID, event.Repositories)
		if err := models.ReplaceVCSProviderRepositories(tx, models.ProviderGitHub, installationID, repositories); err != nil {
			return err
		}
		return enqueueWebhookRepositories(tx, repositories)

	case *gh.InstallationRepositoriesEvent:
		if event.GetInstallation() != nil {
			installation := githubInstallationModel(event.GetInstallation())
			installation.RepositorySelection = event.GetRepositorySelection()
			if err := models.UpsertVCSProviderInstallation(tx, &installation); err != nil {
				return err
			}
			if err := models.DeleteVCSProviderInstallRequestsForAccount(tx, models.ProviderGitHub, installation.AccountID, installation.AccountLogin); err != nil {
				return err
			}
		}
		added := githubRepositoryModels(installationID, event.RepositoriesAdded)
		if err := models.UpsertVCSProviderRepositories(tx, models.ProviderGitHub, installationID, added); err != nil {
			return err
		}
		removedIDs := make([]int64, 0, len(event.RepositoriesRemoved))
		for _, repository := range event.RepositoriesRemoved {
			if repository != nil && repository.GetID() > 0 {
				removedIDs = append(removedIDs, repository.GetID())
			}
		}
		if err := models.DeleteVCSProviderRepositories(tx, models.ProviderGitHub, installationID, removedIDs); err != nil {
			return err
		}
		// The added and removed lists can be empty or truncated, including
		// when access changes to every repository. Replace the catalog from
		// the installation API.
		if err := models.EnqueueVCSProviderInstallationReconciliation(
			tx,
			models.ProviderGitHub,
			installationID,
			uuid.Nil,
			time.Now(),
		); err != nil {
			return err
		}
		return enqueueWebhookRepositories(tx, added)

	case *gh.MemberEvent:
		if event.GetRepo().GetID() <= 0 {
			return nil
		}
		if _, err := models.FindVCSProviderRepository(tx, models.ProviderGitHub, event.GetRepo().GetID()); errors.Is(err, gorm.ErrRecordNotFound) {
			return models.EnqueueVCSProviderReconciliation(tx, models.ProviderGitHub, time.Now())
		} else if err != nil {
			return err
		}
		return models.DelayVCSProviderRepositorySync(
			tx,
			models.ProviderGitHub,
			event.GetRepo().GetID(),
			time.Now().Add(githubRepositoryAccessChangeDelay),
			models.VCSProviderRepositorySyncPriorityInteractive,
		)

	case *gh.TeamAddEvent:
		return delayGitHubRepositoryAccessSync(tx, event.GetRepo())

	case *gh.TeamEvent:
		if event.GetRepo().GetID() > 0 {
			return delayGitHubRepositoryAccessSync(tx, event.GetRepo())
		}
		if event.GetAction() == "deleted" {
			return delayGitHubInstallationAccessSync(tx, installationID)
		}
		return nil

	case *gh.MembershipEvent:
		return delayGitHubInstallationAccessSync(tx, installationID)

	case *gh.OrganizationEvent:
		switch event.GetAction() {
		case "member_added", "member_removed":
			return delayGitHubInstallationAccessSync(tx, installationID)
		}
		return nil
	}

	return nil
}

// GitHub can take a few seconds to publish access changes through the
// collaborators API, so access syncs wait before they run.
const (
	githubRepositoryAccessChangeDelay   = 10 * time.Second
	githubInstallationAccessChangeDelay = 30 * time.Second
)

func delayGitHubRepositoryAccessSync(tx *gorm.DB, repository *gh.Repository) error {
	if repository.GetID() <= 0 {
		return nil
	}
	if _, err := models.FindVCSProviderRepository(tx, models.ProviderGitHub, repository.GetID()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return models.DelayVCSProviderRepositorySync(
		tx,
		models.ProviderGitHub,
		repository.GetID(),
		time.Now().Add(githubRepositoryAccessChangeDelay),
		models.VCSProviderRepositorySyncPriorityInteractive,
	)
}

// Team and organization membership changes do not say which repositories
// they affect, so every repository of the installation is refreshed. These
// syncs use background priority, so a large organization does not delay
// onboarding refreshes.
func delayGitHubInstallationAccessSync(tx *gorm.DB, installationID int64) error {
	return models.DelayVCSProviderInstallationRepositorySyncs(
		tx,
		models.ProviderGitHub,
		installationID,
		time.Now().Add(githubInstallationAccessChangeDelay),
		models.VCSProviderRepositorySyncPriorityBackground,
	)
}

func enqueueWebhookRepositories(tx *gorm.DB, repositories []models.VCSProviderRepository) error {
	runAt := time.Now()
	for _, repository := range repositories {
		if err := models.EnqueueVCSProviderRepositorySync(
			tx,
			models.ProviderGitHub,
			repository.RepositoryID,
			runAt,
			models.VCSProviderRepositorySyncPriorityInteractive,
		); err != nil {
			return err
		}
	}
	return nil
}

func githubInstallationID(payload []byte) (int64, bool) {
	var envelope struct {
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Installation.ID <= 0 {
		return 0, false
	}
	return envelope.Installation.ID, true
}

func writeHostedGitHubAppAuthError(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusUnauthorized:
		http.Error(w, "Unauthorized", status)
	case http.StatusForbidden:
		http.Error(w, "Forbidden", status)
	default:
		http.Error(w, http.StatusText(status), status)
	}
}

func githubInstallationModel(installation *gh.Installation) models.VCSProviderInstallation {
	var accountID *int64
	if installation.GetAccount().GetID() > 0 {
		value := installation.GetAccount().GetID()
		accountID = &value
	}
	var suspendedAt *time.Time
	if installation.SuspendedAt != nil {
		value := installation.SuspendedAt.Time
		suspendedAt = &value
	}
	return models.VCSProviderInstallation{
		Provider:            models.ProviderGitHub,
		InstallationID:      installation.GetID(),
		AccountID:           accountID,
		AccountLogin:        installation.GetAccount().GetLogin(),
		AccountType:         installation.GetTargetType(),
		HTMLURL:             installation.GetHTMLURL(),
		RepositorySelection: installation.GetRepositorySelection(),
		SuspendedAt:         suspendedAt,
	}
}

func githubRepositoryModels(installationID int64, repositories []*gh.Repository) []models.VCSProviderRepository {
	result := make([]models.VCSProviderRepository, 0, len(repositories))
	for _, repository := range repositories {
		if repository == nil || repository.GetID() <= 0 || repository.GetFullName() == "" {
			continue
		}
		result = append(result, models.VCSProviderRepository{
			RepositoryID:   repository.GetID(),
			InstallationID: installationID,
			FullName:       repository.GetFullName(),
			Private:        repository.GetPrivate(),
			DefaultBranch:  repository.GetDefaultBranch(),
		})
	}
	return result
}
