package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type installationLLMSettingsResponse struct {
	WelcomeGrantCents     int64                       `json:"welcome_grant_cents"`
	MarkupBPS             int                         `json:"markup_bps"`
	WarningThresholdBPS   int                         `json:"warning_threshold_bps"`
	DefaultHostedProvider string                      `json:"default_hosted_provider"`
	DefaultHostedModel    string                      `json:"default_hosted_model"`
	Providers             []hostedLLMProviderResponse `json:"providers"`
}

type hostedLLMProviderResponse struct {
	Provider                string   `json:"provider"`
	Enabled                 bool     `json:"enabled"`
	APIKeyConfigured        bool     `json:"api_key_configured"`
	ManagementKeyConfigured bool     `json:"management_key_configured"`
	BaseURL                 string   `json:"base_url"`
	AllowedModels           []string `json:"allowed_models"`
}

type installationLLMSettingsRequest struct {
	WelcomeGrantCents     *int64  `json:"welcome_grant_cents"`
	MarkupBPS             *int    `json:"markup_bps"`
	WarningThresholdBPS   *int    `json:"warning_threshold_bps"`
	DefaultHostedProvider *string `json:"default_hosted_provider"`
	DefaultHostedModel    *string `json:"default_hosted_model"`
}

type hostedLLMProviderRequest struct {
	Enabled       *bool    `json:"enabled"`
	APIKey        *string  `json:"api_key"`
	ManagementKey *string  `json:"management_key"`
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
	RemainingCreditCents    int64   `json:"remaining_credit_cents"`
	GrantTotalCents         int64   `json:"grant_total_cents"`
	SuperplaneGrantCents    int64   `json:"superplane_grant_cents"`
	PurchasedCreditCents    int64   `json:"purchased_credit_cents"`
	HostedBilledCents       int64   `json:"hosted_billed_cents"`
	WelcomeRemainingCents   int64   `json:"welcome_remaining_cents"`
	IncludedRemainingCents  int64   `json:"included_remaining_cents"`
	PurchasedRemainingCents int64   `json:"purchased_remaining_cents"`
	AdminRemainingCents     int64   `json:"admin_remaining_cents"`
	WelcomeCreditExpiresAt  *string `json:"welcome_credit_expires_at"`
	MarkupBPS               int     `json:"markup_bps"`
	MarkupOverrideBPS       *int    `json:"markup_override_bps"`
	Warning                 bool    `json:"warning"`
}

type addOrganizationLLMCreditRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
}

type setOrganizationCreditBalanceRequest struct {
	ExpirationChoice       *string    `json:"expiration"`
	ExpiresAt              *time.Time `json:"expires_at"`
	Bucket                 string     `json:"bucket"`
	TargetCents            int64      `json:"target_cents"`
	ExpectedRemainingCents int64      `json:"expected_remaining_cents"`
	Note                   string     `json:"note"`
}

// organizationCreditGrantResponse matches the OrganizationCreditGrant JSON
// shape from the organization API so the UI renders both with one table.
type organizationCreditGrantResponse struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	AmountCents  int64   `json:"amountCents,string"`
	Note         string  `json:"note"`
	ActorName    string  `json:"actorName,omitempty"`
	PolarOrderID string  `json:"polarOrderId,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	ExpiresAt    *string `json:"expiresAt,omitempty"`
}

type listOrganizationCreditGrantsResponse struct {
	Grants []organizationCreditGrantResponse `json:"grants"`
}

type organizationLLMMarkupRequest struct {
	MarkupBPS *int `json:"markup_bps"`
}

type organizationBillingPlanResponse struct {
	CreditExpiration              string    `json:"credit_expiration"`
	CreditValidityDays            *int      `json:"credit_validity_days"`
	CatalogID                     uuid.UUID `json:"catalog_id"`
	CatalogKey                    string    `json:"catalog_key"`
	CatalogName                   string    `json:"catalog_name"`
	CatalogCategory               string    `json:"catalog_category"`
	Plan                          string    `json:"plan"`
	PlanSource                    string    `json:"plan_source"`
	PolarSubscriptionVerification string    `json:"polar_subscription_verification"`
	PolarSubscriptionID           string    `json:"polar_subscription_id"`
	PolarSubscriptionURL          string    `json:"polar_subscription_url,omitempty"`
	PolarSubscriptionStatus       string    `json:"polar_subscription_status"`
	PolarManaged                  bool      `json:"polar_managed"`
	TrialEndsAt                   *string   `json:"trial_ends_at"`
	CurrentPeriodEnd              *string   `json:"current_period_end"`
}

type organizationBillingPlanRequest struct {
	Plan        string  `json:"plan"`
	TrialEndsAt *string `json:"trial_ends_at"`
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

	if req.WelcomeGrantCents != nil {
		http.Error(w, "Edit trial credit in Billing plans", http.StatusBadRequest)
		return
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		current, err := models.GetInstallationLLMSettings(tx)
		if err != nil {
			return err
		}
		next := *current
		if req.MarkupBPS != nil {
			next.MarkupBPS = *req.MarkupBPS
		}
		if req.WarningThresholdBPS != nil {
			next.WarningThresholdBPS = *req.WarningThresholdBPS
		}
		if req.DefaultHostedProvider != nil {
			next.DefaultHostedProvider = optionalTrimmedStringPointer(*req.DefaultHostedProvider)
		}
		if req.DefaultHostedModel != nil {
			next.DefaultHostedModel = optionalTrimmedStringPointer(*req.DefaultHostedModel)
		}
		_, err = models.UpdateInstallationLLMSettings(tx, next)
		return err
	})
	if err != nil {
		status := http.StatusBadRequest
		if !isClientLLMSettingsError(err) {
			log.Errorf("admin: failed to update hosted LLM settings: %v", err)
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
		if req.ManagementKey != nil {
			key := strings.TrimSpace(*req.ManagementKey)
			if key == "" {
				next.ManagementKey = nil
			} else {
				if provider == models.UsageProviderOpenRouter {
					client := openrouter.NewManagementClient(s.registry.HTTPContext(), key)
					if verifyErr := client.VerifyManagement(); verifyErr != nil {
						return fmt.Errorf("provisioning API key is invalid: %w", verifyErr)
					}
				}
				encrypted, encryptErr := llm.EncryptManagementKey(r.Context(), s.encryptor, provider, key)
				if encryptErr != nil {
					return encryptErr
				}
				next.ManagementKey = encrypted
			}
		}
		if next.Enabled && !next.HasAPIKey() {
			return errors.New("API key is required when the provider is enabled")
		}
		if next.Enabled && provider == models.UsageProviderOpenRouter && !next.HasManagementKey() {
			return errors.New("provisioning API key is required when OpenRouter is enabled")
		}
		if next.Enabled && len(next.AllowedModels) == 0 {
			return errors.New("select at least one model when the provider is enabled")
		}
		if _, err = models.UpsertHostedLLMProvider(tx, next); err != nil {
			return err
		}
		return models.SyncDefaultHostedLLMModelAfterProviderChange(tx)
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

func (s *Server) adminListOrganizationCreditGrants(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	db := database.DB(r.Context())
	grants, err := models.ListOrganizationLLMCreditGrants(db, orgID)
	if err != nil {
		log.Errorf("admin: failed to list organization credit grants: %v", err)
		http.Error(w, "Failed to load credit history", http.StatusInternalServerError)
		return
	}
	actorNames, err := models.CreditGrantActorNames(db, grants)
	if err != nil {
		log.Errorf("admin: failed to load credit grant actors: %v", err)
		http.Error(w, "Failed to load credit history", http.StatusInternalServerError)
		return
	}

	out := make([]organizationCreditGrantResponse, 0, len(grants))
	for _, grant := range grants {
		out = append(out, serializeAdminCreditGrant(grant, actorNames))
	}
	respondJSON(w, listOrganizationCreditGrantsResponse{Grants: out})
}

func (s *Server) adminSetOrganizationCreditBalance(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	var req setOrganizationCreditBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	account, _ := middleware.GetAccountFromContext(r.Context())
	var actor *uuid.UUID
	if account != nil {
		actor = &account.ID
	}

	targetMicros, err := creditCentsToMicros(req.TargetCents)
	if err != nil {
		writeCreditBalanceError(w, err)
		return
	}
	expectedMicros, err := creditCentsToMicros(req.ExpectedRemainingCents)
	if err != nil {
		writeCreditBalanceError(w, err)
		return
	}

	db := database.DB(r.Context())
	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID:   orgID,
		ExpirationChoice: req.ExpirationChoice,
		ExpiresAt:        req.ExpiresAt,
		Bucket:           req.Bucket,
		TargetMicros:     targetMicros,
		ExpectedMicros:   expectedMicros,
		Note:             req.Note,
		ActorAccountID:   actor,
	})
	if err != nil {
		writeCreditBalanceError(w, err)
		return
	}

	response, err := describeOrganizationLLMCreditJSON(db, orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization LLM credit: %v", err)
		http.Error(w, "Failed to load organization credit", http.StatusInternalServerError)
		return
	}
	respondJSON(w, response)
}

var errCreditCentsOutOfRange = errors.New("credit balance is out of range")

func creditCentsToMicros(cents int64) (int64, error) {
	if cents < 0 {
		return 0, models.ErrCreditBalanceNegative
	}
	if cents > math.MaxInt64/models.MicrosPerCent {
		return 0, errCreditCentsOutOfRange
	}
	return cents * models.MicrosPerCent, nil
}

func writeCreditBalanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrCreditBalanceChanged):
		http.Error(w, "The balance changed. Reload and try again.", http.StatusConflict)
	case errors.Is(err, models.ErrCreditBalanceBucket),
		errors.Is(err, models.ErrCreditExpiration),
		errors.Is(err, models.ErrCreditBalanceNegative),
		errors.Is(err, errCreditCentsOutOfRange):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrTrialCreditNotActive):
		http.Error(w, "This organization has no active trial credit.", http.StatusBadRequest)
	default:
		log.Errorf("admin: failed to set organization credit balance: %v", err)
		http.Error(w, "Failed to set credit balance", http.StatusInternalServerError)
	}
}

func serializeAdminCreditGrant(grant models.OrganizationLLMCreditGrant, actorNames map[uuid.UUID]string) organizationCreditGrantResponse {
	item := organizationCreditGrantResponse{
		ID:          grant.ID.String(),
		Kind:        grant.Kind,
		AmountCents: models.SignedMicrosToCents(grant.AmountMicros),
		Note:        grant.Note,
		CreatedAt:   grant.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:   formatOptionalTime(grant.ExpiresAt),
	}
	if grant.ActorAccountID != nil {
		item.ActorName = actorNames[*grant.ActorAccountID]
	}
	if grant.PolarOrderID != nil {
		item.PolarOrderID = *grant.PolarOrderID
	}
	return item
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

// adminBillingPlanListItem is an admin list response item. It is not a
// persisted model and the reason is never accepted as deletion authorization.
type adminBillingPlanListItem struct {
	models.BillingPlan
	DeleteDisabledReason *string `json:"delete_disabled_reason"`
}

func (s *Server) adminListBillingPlans(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	plans, err := models.ListBillingPlans(db)
	if err != nil {
		http.Error(w, "Failed to load billing plans", http.StatusInternalServerError)
		return
	}
	planIDs := make([]uuid.UUID, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID)
	}
	assigned, err := models.ListAssignedBillingPlanIDs(db, planIDs)
	if err != nil {
		http.Error(w, "Failed to load billing plans", http.StatusInternalServerError)
		return
	}
	items := make([]adminBillingPlanListItem, 0, len(plans))
	for _, plan := range plans {
		item := adminBillingPlanListItem{BillingPlan: plan}
		if reason := plan.DeleteBlockReason(assigned.Contains(plan.ID)); reason != "" {
			item.DeleteDisabledReason = &reason
		}
		items = append(items, item)
	}
	respondJSON(w, items)
}

func (s *Server) adminGetOrganizationBillingPlan(w http.ResponseWriter, r *http.Request) {
	if !models.BillingEnabled() {
		http.Error(w, "Billing is not available", http.StatusNotFound)
		return
	}
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	response, err := describeOrganizationBillingPlanJSON(database.Conn(), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization billing plan: %v", err)
		http.Error(w, "Failed to load organization billing plan", http.StatusInternalServerError)
		return
	}
	verification, verifyErr := polar.VerifyOrganizationSubscription(r.Context(), orgID)
	if verifyErr != nil {
		log.WithError(verifyErr).Warn("admin: failed to verify Polar subscription")
	}
	response.PolarSubscriptionVerification = verification
	respondJSON(w, response)
}

func (s *Server) adminSetOrganizationBillingPlan(w http.ResponseWriter, r *http.Request) {
	if !models.BillingEnabled() {
		http.Error(w, "Billing is not available", http.StatusNotFound)
		return
	}
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	var req organizationBillingPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	snapshot, err := models.ResolveOrganizationBillingPlan(db, orgID)
	if err != nil {
		http.Error(w, "Failed to load organization billing plan", http.StatusInternalServerError)
		return
	}
	verification, err := polar.VerifyOrganizationSubscription(r.Context(), orgID)
	if err != nil {
		log.WithError(err).Warn("admin: failed to verify Polar subscription")
		http.Error(w, "SuperPlane could not verify the Polar subscription. Refresh and try again.", http.StatusBadGateway)
		return
	}
	if verification == "active" {
		http.Error(w, models.ErrPolarManagedBillingPlan.Error(), http.StatusBadRequest)
		return
	}

	planName := strings.TrimSpace(req.Plan)
	trialEndsAt, err := trialEndForAdminPlan(database.Conn(), planName, req.TrialEndsAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if verification == "none" {
		_, err = snapshot.AssignAdminPlanAfterPolarVerification(db, planName, trialEndsAt)
	} else {
		_, err = models.SetAdminOrganizationPlan(db, orgID, planName, trialEndsAt)
	}
	if errors.Is(err, models.ErrBillingPlanChanged) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := describeOrganizationBillingPlanJSON(database.Conn(), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization billing plan: %v", err)
		http.Error(w, "Failed to load organization billing plan", http.StatusInternalServerError)
		return
	}
	response.PolarSubscriptionVerification = verification
	respondJSON(w, response)
}

func describeOrganizationBillingPlanJSON(tx *gorm.DB, orgID uuid.UUID) (organizationBillingPlanResponse, error) {
	plan, err := models.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return organizationBillingPlanResponse{}, err
	}
	subscriptionID := ""
	if plan.PolarSubscriptionID != nil {
		subscriptionID = *plan.PolarSubscriptionID
	}
	return organizationBillingPlanResponse{
		PolarSubscriptionID:     subscriptionID,
		PolarSubscriptionURL:    polar.SubscriptionDashboardURL(subscriptionID),
		CatalogID:               plan.PlanID,
		CatalogKey:              plan.Catalog.Key,
		CatalogName:             plan.Catalog.Name,
		CatalogCategory:         plan.Catalog.Category,
		CreditExpiration:        plan.Catalog.CreditExpiration,
		CreditValidityDays:      plan.Catalog.CreditValidityDays,
		Plan:                    plan.Plan,
		PlanSource:              plan.PlanSource,
		PolarSubscriptionStatus: plan.PolarSubscriptionStatus,
		PolarManaged:            models.OrganizationBillingIsPolarManaged(plan),
		TrialEndsAt:             formatOptionalTime(plan.TrialEndsAt),
		CurrentPeriodEnd:        formatOptionalTime(plan.CurrentPeriodEnd),
	}, nil
}

func trialEndForAdminPlan(tx *gorm.DB, planName string, raw *string) (*time.Time, error) {
	catalog, err := models.FindBillingPlan(tx, planName)
	if err != nil || catalog.Category != "trial" {
		return nil, nil
	}
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, errors.New("trial end date is required")
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, errors.New("trial end date is required")
	}
	if !parsed.After(time.Now()) {
		return nil, errors.New("choose a future date")
	}
	end := parsed.UTC()
	return &end, nil
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
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
			Provider:                name,
			Enabled:                 row.Enabled,
			APIKeyConfigured:        row.HasAPIKey(),
			ManagementKeyConfigured: row.HasManagementKey(),
			BaseURL:                 row.BaseURL,
			AllowedModels:           append([]string{}, row.AllowedModels...),
		})
	}

	catalog, err := models.FindBillingPlan(tx, models.BillingPlanTrial)
	if err != nil {
		return installationLLMSettingsResponse{}, err
	}
	defaultModel := models.InstallationDefaultHostedLLMModel(settings)
	return installationLLMSettingsResponse{
		WelcomeGrantCents:     catalog.TrialCreditCents,
		MarkupBPS:             settings.MarkupBPS,
		WarningThresholdBPS:   settings.WarningThresholdBPS,
		DefaultHostedProvider: defaultModel.Provider,
		DefaultHostedModel:    defaultModel.Model,
		Providers:             providers,
	}, nil
}

func optionalTrimmedStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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
		RemainingCreditCents:    pricebook.MicrosToCents(summary.RemainingMicros),
		GrantTotalCents:         pricebook.MicrosToCents(summary.GrantMicros),
		SuperplaneGrantCents:    pricebook.MicrosToCents(summary.SuperPlaneGrantMicros),
		PurchasedCreditCents:    pricebook.MicrosToCents(summary.PurchasedCreditMicros),
		HostedBilledCents:       pricebook.MicrosToCents(summary.BilledMicros),
		WelcomeRemainingCents:   pricebook.MicrosToCents(summary.WelcomeRemainingMicros),
		IncludedRemainingCents:  pricebook.MicrosToCents(summary.IncludedRemainingMicros),
		PurchasedRemainingCents: pricebook.MicrosToCents(summary.PurchasedRemainingMicros),
		AdminRemainingCents:     pricebook.MicrosToCents(summary.AdminRemainingMicros),
		WelcomeCreditExpiresAt:  formatOptionalTime(summary.WelcomeCreditExpiresAt),
		MarkupBPS:               summary.MarkupBPS,
		MarkupOverrideBPS:       override,
		Warning:                 summary.Warning,
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
	if errors.Is(err, models.ErrHostedLLMProviderNotFound) ||
		errors.Is(err, models.ErrDefaultHostedModelIncomplete) ||
		errors.Is(err, models.ErrDefaultHostedModelNotOnAllowlist) ||
		errors.Is(err, models.ErrDefaultHostedModelMustBeReplaced) {
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
		strings.Contains(msg, "provisioning api key") ||
		strings.Contains(msg, "llm base url") ||
		strings.Contains(msg, "superplane agent model")
}
