package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	appcatalog "github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const githubInstallApprovedPath = "/github/approved"

var enqueueGitHubAppReconciliation = func(ctx context.Context, availableAt time.Time) error {
	return models.EnqueueVCSProviderReconciliation(database.DB(ctx), models.ProviderGitHub, availableAt)
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

var refreshGitHubAppInstallRequests = func(ctx context.Context) error {
	catalog, err := appcatalog.NewCatalog(database.DB(ctx), config.LoadGitHubHostedAppConfig())
	if err != nil {
		return err
	}
	return catalog.ReconcileInstallRequests(ctx)
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

	catalog, err := appcatalog.NewCatalog(db, config.LoadGitHubHostedAppConfig())
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
	organizationID := githubAppSetupOrganizationID(query.Get("state"))

	switch query.Get("setup_action") {
	case "install":
		redirectPath := "/"
		if strings.TrimSpace(query.Get("state")) == "" {
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
	case "update":
		if err := enqueueGitHubAppInstallationReconciliation(r.Context(), installationID, organizationID, time.Now()); err != nil {
			http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	default:
		http.Error(w, "invalid setup action", http.StatusBadRequest)
	}
}

// The user returns to the wizard right after this redirect, so the request
// list is read now. The queued reconciliation is only the fallback.
func (s *Server) handleGitHubAppInstallRequest(w http.ResponseWriter, r *http.Request) {
	err := refreshGitHubAppInstallRequests(r.Context())
	if err == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.WithError(err).Warn("failed to refresh GitHub App installation requests")
	if err := enqueueGitHubAppReconciliation(r.Context(), time.Now()); err != nil {
		http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func githubAppSetupOrganizationID(state string) uuid.UUID {
	cfg := config.LoadGitHubHostedAppConfig()
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
	app, ok := common.HostedAppFromEnv()
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
			time.Now().Add(10*time.Second),
			models.VCSProviderRepositorySyncPriorityInteractive,
		)
	}

	return nil
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
