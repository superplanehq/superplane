package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type installationLLMSettingsResponse struct {
	WelcomeGrantCents   int64                       `json:"welcome_grant_cents"`
	MarkupBPS           int                         `json:"markup_bps"`
	WarningThresholdBPS int                         `json:"warning_threshold_bps"`
	Providers           []hostedLLMProviderResponse `json:"providers"`
}

type hostedLLMProviderResponse struct {
	Provider         string   `json:"provider"`
	Enabled          bool     `json:"enabled"`
	APIKeyConfigured bool     `json:"api_key_configured"`
	BaseURL          string   `json:"base_url"`
	AllowedModels    []string `json:"allowed_models"`
}

type installationLLMSettingsRequest struct {
	WelcomeGrantCents   *int64 `json:"welcome_grant_cents"`
	MarkupBPS           *int   `json:"markup_bps"`
	WarningThresholdBPS *int   `json:"warning_threshold_bps"`
}

type hostedLLMProviderRequest struct {
	Enabled       *bool    `json:"enabled"`
	APIKey        *string  `json:"api_key"`
	BaseURL       *string  `json:"base_url"`
	AllowedModels []string `json:"allowed_models"`
}

type listHostedLLMModelsRequest struct {
	APIKey  *string `json:"api_key"`
	BaseURL *string `json:"base_url"`
}

type listHostedLLMModelsResponse struct {
	Models []hostedLLMModelResponse `json:"models"`
}

type hostedLLMModelResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type organizationLLMCreditResponse struct {
	RemainingCreditCents int64 `json:"remaining_credit_cents"`
	GrantTotalCents      int64 `json:"grant_total_cents"`
	HostedBilledCents    int64 `json:"hosted_billed_cents"`
	MarkupBPS            int   `json:"markup_bps"`
	MarkupOverrideBPS    *int  `json:"markup_override_bps"`
	Warning              bool  `json:"warning"`
}

type addOrganizationLLMCreditRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
}

type organizationLLMMarkupRequest struct {
	MarkupBPS *int `json:"markup_bps"`
}

func (s *Server) adminGetInstallationLLMSettings(w http.ResponseWriter, r *http.Request) {
	response, err := s.buildInstallationLLMSettingsResponse()
	if err != nil {
		log.Errorf("admin: failed to load hosted LLM settings: %v", err)
		http.Error(w, "Failed to load hosted LLM settings", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) adminUpdateInstallationLLMSettings(w http.ResponseWriter, r *http.Request) {
	var req installationLLMSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		current, err := models.GetInstallationLLMSettings(tx)
		if err != nil {
			return err
		}
		next := *current
		if req.WelcomeGrantCents != nil {
			next.WelcomeGrantCents = *req.WelcomeGrantCents
		}
		if req.MarkupBPS != nil {
			next.MarkupBPS = *req.MarkupBPS
		}
		if req.WarningThresholdBPS != nil {
			next.WarningThresholdBPS = *req.WarningThresholdBPS
		}
		_, err = models.UpdateInstallationLLMSettings(tx, next)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.buildInstallationLLMSettingsResponse()
	if err != nil {
		log.Errorf("admin: failed to load hosted LLM settings: %v", err)
		http.Error(w, "Failed to load hosted LLM settings", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) adminUpdateHostedLLMProvider(w http.ResponseWriter, r *http.Request) {
	provider, err := models.NormalizeHostedLLMProvider(mux.Vars(r)["provider"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req hostedLLMProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		current, err := models.FindHostedLLMProvider(tx, provider)
		if err != nil && !errors.Is(err, models.ErrHostedLLMProviderNotFound) {
			return err
		}

		next := models.HostedLLMProvider{Provider: provider}
		if current != nil {
			next = *current
		}
		if req.Enabled != nil {
			next.Enabled = *req.Enabled
		}
		if req.BaseURL != nil {
			next.BaseURL = strings.TrimSpace(*req.BaseURL)
			if err := llm.ValidateBaseURL(next.BaseURL); err != nil {
				return err
			}
		}
		if req.AllowedModels != nil {
			next.AllowedModels = datatypes.JSONSlice[string](req.AllowedModels)
		}
		if req.APIKey != nil {
			key := strings.TrimSpace(*req.APIKey)
			if key == "" {
				next.APIKey = nil
			} else {
				encrypted, encryptErr := llm.EncryptAPIKey(r.Context(), s.encryptor, provider, key)
				if encryptErr != nil {
					return encryptErr
				}
				next.APIKey = encrypted
			}
		}
		if next.Enabled && !next.HasAPIKey() {
			return errors.New("API key is required when the provider is enabled")
		}
		if next.Enabled && len(next.AllowedModels) == 0 {
			return errors.New("select at least one model when the provider is enabled")
		}
		_, err = models.UpsertHostedLLMProvider(tx, next)
		return err
	})
	if err != nil {
		status := http.StatusBadRequest
		if !isClientLLMSettingsError(err) {
			log.Errorf("admin: failed to update hosted LLM provider %s: %v", provider, err)
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	response, err := s.buildInstallationLLMSettingsResponse()
	if err != nil {
		log.Errorf("admin: failed to load hosted LLM settings: %v", err)
		http.Error(w, "Failed to load hosted LLM settings", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) adminListHostedLLMProviderModels(w http.ResponseWriter, r *http.Request) {
	provider, err := models.NormalizeHostedLLMProvider(mux.Vars(r)["provider"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req listHostedLLMModelsRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	apiKey := ""
	storedBaseURL := ""
	if req.APIKey != nil {
		apiKey = strings.TrimSpace(*req.APIKey)
	}

	if apiKey == "" {
		row, findErr := models.FindHostedLLMProvider(database.Conn(), provider)
		if findErr != nil {
			http.Error(w, "Save an API key before you list models", http.StatusBadRequest)
			return
		}
		decrypted, decryptErr := llm.DecryptAPIKey(r.Context(), s.encryptor, provider, row.APIKey)
		if decryptErr != nil {
			http.Error(w, "Save an API key before you list models", http.StatusBadRequest)
			return
		}
		apiKey = decrypted
		storedBaseURL = row.BaseURL
	}
	baseURL := resolveHostedListModelsBaseURL(req.BaseURL, storedBaseURL)
	if err := llm.ValidateBaseURL(baseURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := llm.New(s.registry.HTTPContext(), provider, llm.Credentials{APIKey: apiKey, BaseURL: baseURL})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modelsList, err := client.ListModels(r.Context())
	if err != nil {
		http.Error(w, "Unable to list models from the provider", http.StatusBadGateway)
		return
	}

	out := listHostedLLMModelsResponse{Models: make([]hostedLLMModelResponse, 0, len(modelsList))}
	for _, model := range modelsList {
		out.Models = append(out.Models, hostedLLMModelResponse{ID: model.ID, Name: model.ID})
	}
	respondJSON(w, out)
}

func (s *Server) adminGetOrganizationLLMCredit(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	response, err := describeOrganizationLLMCreditJSON(database.Conn(), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization LLM credit: %v", err)
		http.Error(w, "Failed to load organization credit", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) adminAddOrganizationLLMCredit(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	var req addOrganizationLLMCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	account, _ := middleware.GetAccountFromContext(r.Context())
	var actor *uuid.UUID
	if account != nil {
		actor = &account.ID
	}

	_, err := models.AddAdminLLMCreditGrant(database.Conn(), orgID, models.CentsToMicros(req.AmountCents), req.Note, actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := describeOrganizationLLMCreditJSON(database.Conn(), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization LLM credit: %v", err)
		http.Error(w, "Failed to load organization credit", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) adminUpdateOrganizationLLMMarkup(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	var req organizationLLMMarkupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.UpsertOrganizationLLMMarkup(database.Conn(), orgID, req.MarkupBPS); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := describeOrganizationLLMCreditJSON(database.Conn(), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization LLM credit: %v", err)
		http.Error(w, "Failed to load organization credit", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

func (s *Server) buildInstallationLLMSettingsResponse() (installationLLMSettingsResponse, error) {
	tx := database.Conn()
	settings, err := models.GetInstallationLLMSettings(tx)
	if err != nil {
		return installationLLMSettingsResponse{}, err
	}

	stored, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		return installationLLMSettingsResponse{}, err
	}
	byProvider := map[string]models.HostedLLMProvider{}
	for _, row := range stored {
		byProvider[row.Provider] = row
	}

	providers := make([]hostedLLMProviderResponse, 0, len(models.KnownHostedLLMProviders()))
	for _, name := range models.KnownHostedLLMProviders() {
		row := byProvider[name]
		providers = append(providers, hostedLLMProviderResponse{
			Provider:         name,
			Enabled:          row.Enabled,
			APIKeyConfigured: row.HasAPIKey(),
			BaseURL:          row.BaseURL,
			AllowedModels:    append([]string{}, row.AllowedModels...),
		})
	}

	return installationLLMSettingsResponse{
		WelcomeGrantCents:   settings.WelcomeGrantCents,
		MarkupBPS:           settings.MarkupBPS,
		WarningThresholdBPS: settings.WarningThresholdBPS,
		Providers:           providers,
	}, nil
}

func describeOrganizationLLMCreditJSON(tx *gorm.DB, orgID uuid.UUID) (organizationLLMCreditResponse, error) {
	summary, err := models.DescribeOrganizationLLMCredit(tx, orgID)
	if err != nil {
		return organizationLLMCreditResponse{}, err
	}
	orgSettings, err := models.FindOrganizationLLMSettings(tx, orgID)
	if err != nil {
		return organizationLLMCreditResponse{}, err
	}
	var override *int
	if orgSettings != nil {
		override = orgSettings.MarkupBPS
	}
	return organizationLLMCreditResponse{
		RemainingCreditCents: pricebook.MicrosToCents(summary.RemainingMicros),
		GrantTotalCents:      pricebook.MicrosToCents(summary.GrantMicros),
		HostedBilledCents:    pricebook.MicrosToCents(summary.BilledMicros),
		MarkupBPS:            summary.MarkupBPS,
		MarkupOverrideBPS:    override,
		Warning:              summary.Warning,
	}, nil
}

func parseAdminOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID := mux.Vars(r)["orgId"]
	parsed, err := uuid.Parse(orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return uuid.Nil, false
	}
	if _, err := models.FindOrganizationByID(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return uuid.Nil, false
	}
	return parsed, true
}

func resolveHostedListModelsBaseURL(requestBaseURL *string, storedBaseURL string) string {
	if requestBaseURL != nil {
		return strings.TrimSpace(*requestBaseURL)
	}
	return storedBaseURL
}

func isClientLLMSettingsError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unsupported hosted") ||
		strings.Contains(msg, "api key is required") ||
		strings.Contains(msg, "select at least one model") ||
		strings.Contains(msg, "allowed model") ||
		strings.Contains(msg, "duplicate allowed") ||
		strings.Contains(msg, "markup cannot") ||
		strings.Contains(msg, "welcome grant") ||
		strings.Contains(msg, "warning threshold") ||
		strings.Contains(msg, "llm base url")
}
