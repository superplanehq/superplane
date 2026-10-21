package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

func (s *Server) adminResetBuiltInAutomations(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	result, err := factoryactions.ResetOrganizationBuiltInAutomations(r.Context(), orgID, factoryactions.IntakeDependencies{
		Registry:       s.registry,
		Encryptor:      s.encryptor,
		AuthService:    s.authService,
		WebhookBaseURL: s.WebhooksBaseURL,
	})
	if err != nil {
		log.Errorf("admin: failed to reset built-in automations for org %s: %v", orgID, err)
		var settingsError *factoryactions.BuiltInAutomationSettingsError
		if errors.As(err, &settingsError) {
			http.Error(w, fmt.Sprintf("Reset stopped: %v. No automations were changed. Correct these settings and try again.", err), http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not reset built-in automations.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Errorf("admin: failed to encode built-in automation reset: %v", err)
	}
}

type builtInAutomationListItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	LastResetAt *time.Time `json:"last_reset_at,omitempty"`
	LastResetBy string     `json:"last_reset_by,omitempty"`
}

type builtInAutomationResetResponse struct {
	factoryactions.BuiltInAutomationInstallationReset
	LastResetAt *time.Time `json:"last_reset_at,omitempty"`
	LastResetBy string     `json:"last_reset_by,omitempty"`
}

func (s *Server) adminListBuiltInAutomations(w http.ResponseWriter, r *http.Request) {
	automations, err := describeBuiltInAutomations(database.DB(r.Context()))
	if err != nil {
		log.Errorf("admin: failed to list built-in automations: %v", err)
		http.Error(w, "Could not load built-in automations.", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"automations": automations,
	})
}

func (s *Server) adminResetBuiltInAutomation(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Choose a built-in automation.", http.StatusBadRequest)
		return
	}
	templateID := strings.TrimSpace(request.ID)
	if templateID == "" {
		http.Error(w, "Choose a built-in automation.", http.StatusBadRequest)
		return
	}

	result, err := factoryactions.ResetBuiltInAutomationInAllOrganizations(r.Context(), templateID, factoryactions.IntakeDependencies{
		Registry:       s.registry,
		Encryptor:      s.encryptor,
		AuthService:    s.authService,
		WebhookBaseURL: s.WebhooksBaseURL,
	})
	if err != nil {
		if errors.Is(err, factoryactions.ErrUnknownBuiltInAutomation) {
			http.Error(w, "This automation is not built in.", http.StatusBadRequest)
			return
		}
		log.Errorf("admin: failed to reset built-in automation %s: %v", templateID, err)
		http.Error(w, "Could not reset this automation.", http.StatusInternalServerError)
		return
	}

	response := builtInAutomationResetResponse{BuiltInAutomationInstallationReset: result}
	account, ok := middleware.GetAccountFromContext(r.Context())
	if ok && account != nil {
		resetAt := time.Now().UTC()
		err = models.SaveBuiltInAutomationReset(database.DB(r.Context()), templateID, account.ID, resetAt)
		if err != nil {
			log.Errorf("admin: failed to record built-in automation reset %s: %v", templateID, err)
		} else {
			response.LastResetAt = &resetAt
			response.LastResetBy = accountResetName(*account)
		}
	}

	respondJSON(w, response)
}

func describeBuiltInAutomations(tx *gorm.DB) ([]builtInAutomationListItem, error) {
	resets, err := models.ListBuiltInAutomationResets(tx)
	if err != nil {
		return nil, err
	}

	names, err := builtInAutomationResetNames(tx, resets)
	if err != nil {
		return nil, err
	}

	byTemplate := make(map[string]models.BuiltInAutomationReset, len(resets))
	for _, reset := range resets {
		byTemplate[reset.TemplateID] = reset
	}

	catalog := factoryactions.ListBuiltInAutomations()
	items := make([]builtInAutomationListItem, 0, len(catalog))
	for _, automation := range catalog {
		item := builtInAutomationListItem{ID: automation.ID, Name: automation.Name}
		reset, ok := byTemplate[automation.ID]
		if ok {
			resetAt := reset.ResetAt.UTC()
			item.LastResetAt = &resetAt
			if reset.AccountID != nil {
				item.LastResetBy = names[*reset.AccountID]
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func builtInAutomationResetNames(tx *gorm.DB, resets []models.BuiltInAutomationReset) (map[uuid.UUID]string, error) {
	ids := make([]uuid.UUID, 0, len(resets))
	seen := make(map[uuid.UUID]struct{}, len(resets))
	for _, reset := range resets {
		if reset.AccountID == nil || *reset.AccountID == uuid.Nil {
			continue
		}
		if _, ok := seen[*reset.AccountID]; ok {
			continue
		}
		seen[*reset.AccountID] = struct{}{}
		ids = append(ids, *reset.AccountID)
	}
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	var accounts []models.Account
	err := tx.Unscoped().Select("id", "name", "email").Where("id IN ?", ids).Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	names := make(map[uuid.UUID]string, len(accounts))
	for _, account := range accounts {
		names[account.ID] = accountResetName(account)
	}
	return names, nil
}

func accountResetName(account models.Account) string {
	if name := strings.TrimSpace(account.Name); name != "" {
		return name
	}
	return strings.TrimSpace(account.Email)
}
