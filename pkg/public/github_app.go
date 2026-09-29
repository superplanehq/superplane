package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const githubInstallApprovedPath = "/github/approved"

var enqueueGitHubAppReconciliation = func(ctx context.Context, availableAt time.Time) error {
	return models.EnqueueGitHubAppReconciliation(database.DB(ctx), availableAt)
}

// HandleGitHubAppSetup handles only GitHub's installation and repository
// settings redirect. Signed webhooks are authoritative for catalog state.
func (s *Server) HandleGitHubAppSetup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("setup_action") == "request" {
		if err := enqueueGitHubAppReconciliation(r.Context(), time.Now()); err != nil {
			http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if strings.TrimSpace(query.Get("installation_id")) == "" {
		http.Error(w, "missing installation id", http.StatusBadRequest)
		return
	}

	switch query.Get("setup_action") {
	case "install":
		if err := enqueueGitHubAppReconciliation(r.Context(), time.Now()); err != nil {
			http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, githubInstallApprovedPath, http.StatusFound)
	case "update":
		if err := enqueueGitHubAppReconciliation(r.Context(), time.Now()); err != nil {
			http.Error(w, "failed to queue GitHub App reconciliation", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	default:
		http.Error(w, "invalid setup action", http.StatusBadRequest)
	}
}

// HandleGitHubAppWebhook validates the public App signature, updates the
// global catalog, and fans repository events out to every local binding for
// the installation.
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

	// Load bindings before an uninstall removes the catalog row and its
	// binding references.
	integrations, err := models.ListGitHubAppBoundIntegrations(database.DB(r.Context()), installationID)
	if err != nil {
		log.WithError(err).Error("failed to list GitHub App bindings")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := applyGitHubCatalogWebhook(database.DB(r.Context()), event, installationID); err != nil {
		log.WithError(err).Error("failed to update the GitHub App catalog")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if githubAppRepositoryEvent(event) {
		for i := range integrations {
			if err := s.dispatchGitHubAppWebhook(r, payload, &integrations[i]); err != nil {
				log.WithError(err).WithField("integration_id", integrations[i].ID).Error(
					"failed to dispatch GitHub App webhook",
				)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func githubAppRepositoryEvent(event any) bool {
	switch event.(type) {
	case *gh.InstallationEvent, *gh.InstallationRepositoriesEvent, *gh.MemberEvent:
		return false
	default:
		return true
	}
}

func (s *Server) dispatchGitHubAppWebhook(r *http.Request, payload []byte, integration *models.Integration) error {
	webhooks, err := models.ListIntegrationWebhooks(database.DB(r.Context()), integration.ID)
	if err != nil {
		return fmt.Errorf("list integration webhooks: %w", err)
	}

	dispatchErrors := make([]error, 0)
	for i := range webhooks {
		if webhooks[i].State != models.WebhookStateReady {
			continue
		}
		if !githubAppWebhookMatches(&webhooks[i], r.Header.Get("X-GitHub-Event"), payload) {
			continue
		}

		cloned, err := cloneRequestWithBody(r, payload)
		if err != nil {
			dispatchErrors = append(dispatchErrors, fmt.Errorf("clone webhook request for %s: %w", webhooks[i].ID, err))
			continue
		}
		secret, err := s.encryptor.Decrypt(r.Context(), webhooks[i].Secret, []byte(webhooks[i].ID.String()))
		if err != nil {
			dispatchErrors = append(dispatchErrors, fmt.Errorf("decrypt webhook secret for %s: %w", webhooks[i].ID, err))
			continue
		}
		cloned.Header.Set("X-Hub-Signature-256", "sha256="+crypto.Sign(secret, payload))

		cloned = mux.SetURLVars(cloned, map[string]string{"webhookID": webhooks[i].ID.String()})
		response := httptest.NewRecorder()
		s.HandleWebhook(response, cloned)
		if response.Code >= http.StatusBadRequest {
			dispatchErrors = append(dispatchErrors, fmt.Errorf("webhook %s returned status %d", webhooks[i].ID, response.Code))
		}
	}

	return errors.Join(dispatchErrors...)
}

func githubAppWebhookMatches(webhook *models.Webhook, eventType string, payload []byte) bool {
	var configuration common.WebhookConfiguration
	if webhook == nil || mapstructure.Decode(webhook.Configuration.Data(), &configuration) != nil {
		return false
	}

	eventTypes := configuration.EventTypes
	if len(eventTypes) == 0 && configuration.EventType != "" {
		eventTypes = []string{configuration.EventType}
	}
	if len(eventTypes) > 0 {
		matched := false
		for _, configured := range eventTypes {
			if configured == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	configuredRepository := strings.TrimSpace(configuration.Repository)
	if configuredRepository == "" {
		return true
	}

	var envelope struct {
		Repository struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if json.Unmarshal(payload, &envelope) != nil {
		return false
	}

	return strings.EqualFold(configuredRepository, envelope.Repository.FullName) ||
		strings.EqualFold(configuredRepository, envelope.Repository.Name) ||
		configuredRepository == strconv.FormatInt(envelope.Repository.ID, 10)
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
			return models.DeleteGitHubAppInstallation(tx, installationID)
		}
		if event.GetInstallation() == nil {
			return nil
		}
		installation := githubInstallationModel(event.GetInstallation())
		if err := models.UpsertGitHubAppInstallation(tx, &installation); err != nil {
			return err
		}
		if err := models.DeleteGitHubAppInstallRequestsForAccount(tx, installation.AccountID, installation.AccountLogin); err != nil {
			return err
		}
		if event.GetAction() != "created" {
			return nil
		}
		repositories := githubRepositoryModels(installationID, event.Repositories)
		if err := models.ReplaceGitHubAppRepositories(tx, installationID, repositories); err != nil {
			return err
		}
		return enqueueWebhookRepositories(tx, repositories)

	case *gh.InstallationRepositoriesEvent:
		if event.GetInstallation() != nil {
			installation := githubInstallationModel(event.GetInstallation())
			installation.RepositorySelection = event.GetRepositorySelection()
			if err := models.UpsertGitHubAppInstallation(tx, &installation); err != nil {
				return err
			}
			if err := models.DeleteGitHubAppInstallRequestsForAccount(tx, installation.AccountID, installation.AccountLogin); err != nil {
				return err
			}
		}
		added := githubRepositoryModels(installationID, event.RepositoriesAdded)
		if err := models.UpsertGitHubAppRepositories(tx, installationID, added); err != nil {
			return err
		}
		removedIDs := make([]int64, 0, len(event.RepositoriesRemoved))
		for _, repository := range event.RepositoriesRemoved {
			if repository != nil && repository.GetID() > 0 {
				removedIDs = append(removedIDs, repository.GetID())
			}
		}
		if err := models.DeleteGitHubAppRepositories(tx, installationID, removedIDs); err != nil {
			return err
		}
		return enqueueWebhookRepositories(tx, added)

	case *gh.MemberEvent:
		if event.GetRepo().GetID() <= 0 {
			return nil
		}
		if _, err := models.FindGitHubAppRepository(tx, event.GetRepo().GetID()); errors.Is(err, gorm.ErrRecordNotFound) {
			return models.EnqueueGitHubAppReconciliation(tx, time.Now())
		} else if err != nil {
			return err
		}
		return models.EnqueueGitHubAppRepositorySync(tx, event.GetRepo().GetID(), time.Now().Add(10*time.Second))
	}

	return nil
}

func enqueueWebhookRepositories(tx *gorm.DB, repositories []models.GitHubAppRepository) error {
	runAt := time.Now().Add(10 * time.Second)
	for _, repository := range repositories {
		if err := models.EnqueueGitHubAppRepositorySync(tx, repository.RepositoryID, runAt); err != nil {
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

func cloneRequestWithBody(r *http.Request, body []byte) (*http.Request, error) {
	cloned := r.Clone(r.Context())
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))
	return cloned, nil
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

func githubInstallationModel(installation *gh.Installation) models.GitHubAppInstallation {
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
	return models.GitHubAppInstallation{
		InstallationID:      installation.GetID(),
		AccountID:           accountID,
		AccountLogin:        installation.GetAccount().GetLogin(),
		AccountType:         installation.GetTargetType(),
		HTMLURL:             installation.GetHTMLURL(),
		RepositorySelection: installation.GetRepositorySelection(),
		SuspendedAt:         suspendedAt,
	}
}

func githubRepositoryModels(installationID int64, repositories []*gh.Repository) []models.GitHubAppRepository {
	result := make([]models.GitHubAppRepository, 0, len(repositories))
	for _, repository := range repositories {
		if repository == nil || repository.GetID() <= 0 || repository.GetFullName() == "" {
			continue
		}
		result = append(result, models.GitHubAppRepository{
			RepositoryID:   repository.GetID(),
			InstallationID: installationID,
			FullName:       repository.GetFullName(),
			Private:        repository.GetPrivate(),
			DefaultBranch:  repository.GetDefaultBranch(),
		})
	}
	return result
}
