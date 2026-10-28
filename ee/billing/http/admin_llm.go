package billinghttp

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

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
	RecordExists                  bool                       `json:"record_exists"`
	UpdatedAt                     *string                    `json:"updated_at"`
	PolarCustomerID               string                     `json:"polar_customer_id"`
	PolarCustomerURL              string                     `json:"polar_customer_url,omitempty"`
	PolarCustomerEventsURL        string                     `json:"polar_customer_events_url,omitempty"`
	TrialStartedAt                *string                    `json:"trial_started_at"`
	CurrentPeriodStart            *string                    `json:"current_period_start"`
	Workspaces                    []billingWorkspaceEvidence `json:"workspaces"`
	CheckoutAttempts              []billingCheckoutEvidence  `json:"checkout_attempts"`
	Polar                         polar.BillingDiagnostics   `json:"polar"`
	CreditExpiration              string                     `json:"credit_expiration"`
	CreditValidityDays            *int                       `json:"credit_validity_days"`
	CatalogID                     uuid.UUID                  `json:"catalog_id"`
	CatalogKey                    string                     `json:"catalog_key"`
	CatalogName                   string                     `json:"catalog_name"`
	CatalogCategory               string                     `json:"catalog_category"`
	Plan                          string                     `json:"plan"`
	PlanSource                    string                     `json:"plan_source"`
	PolarSubscriptionVerification string                     `json:"polar_subscription_verification"`
	PolarSubscriptionID           string                     `json:"polar_subscription_id"`
	PolarSubscriptionURL          string                     `json:"polar_subscription_url,omitempty"`
	PolarSubscriptionStatus       string                     `json:"polar_subscription_status"`
	PolarManaged                  bool                       `json:"polar_managed"`
	TrialEndsAt                   *string                    `json:"trial_ends_at"`
	CurrentPeriodEnd              *string                    `json:"current_period_end"`
}

type organizationBillingPlanRequest struct {
	Plan        string  `json:"plan"`
	TrialEndsAt *string `json:"trial_ends_at"`
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

	_, err := billingmodels.AddAdminLLMCreditGrant(database.Conn(), orgID, billingmodels.CentsToMicros(req.AmountCents), req.Note, actor)
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
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, orgID)
	if err != nil {
		log.Errorf("admin: failed to list organization credit grants: %v", err)
		http.Error(w, "Failed to load credit history", http.StatusInternalServerError)
		return
	}
	actorNames, err := billingmodels.CreditGrantActorNames(db, grants)
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
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
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
		return 0, billingmodels.ErrCreditBalanceNegative
	}
	if cents > math.MaxInt64/billingmodels.MicrosPerCent {
		return 0, errCreditCentsOutOfRange
	}
	return cents * billingmodels.MicrosPerCent, nil
}

func writeCreditBalanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, billingmodels.ErrCreditBalanceChanged):
		http.Error(w, "The balance changed. Reload and try again.", http.StatusConflict)
	case errors.Is(err, billingmodels.ErrCreditBalanceBucket),
		errors.Is(err, billingmodels.ErrCreditExpiration),
		errors.Is(err, billingmodels.ErrCreditBalanceNegative),
		errors.Is(err, errCreditCentsOutOfRange):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, billingmodels.ErrTrialCreditNotActive):
		http.Error(w, "This organization has no active trial credit.", http.StatusBadRequest)
	default:
		log.Errorf("admin: failed to set organization credit balance: %v", err)
		http.Error(w, "Failed to set credit balance", http.StatusInternalServerError)
	}
}

func serializeAdminCreditGrant(grant billingmodels.OrganizationLLMCreditGrant, actorNames map[uuid.UUID]string) organizationCreditGrantResponse {
	item := organizationCreditGrantResponse{
		ID:          grant.ID.String(),
		Kind:        grant.Kind,
		AmountCents: billingmodels.SignedMicrosToCents(grant.AmountMicros),
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

	if err := billingmodels.UpsertOrganizationLLMMarkup(database.Conn(), orgID, req.MarkupBPS); err != nil {
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
// persisted model. The display fields are never accepted as authorization:
// the server checks assignments again on save and delete.
type adminBillingPlanListItem struct {
	billingmodels.BillingPlan
	Assigned             bool    `json:"assigned"`
	DeleteDisabledReason *string `json:"delete_disabled_reason"`
}

func (s *Server) adminListBillingPlans(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	plans, err := billingmodels.ListBillingPlans(db)
	if err != nil {
		writeBillingPlanServerError(w, r, err, billingPlanLoadErrorMessage)
		return
	}
	planIDs := make([]uuid.UUID, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID)
	}
	assigned, err := billingmodels.ListAssignedBillingPlanIDs(db, planIDs)
	if err != nil {
		writeBillingPlanServerError(w, r, err, billingPlanLoadErrorMessage)
		return
	}
	items := make([]adminBillingPlanListItem, 0, len(plans))
	for _, plan := range plans {
		item := adminBillingPlanListItem{BillingPlan: plan, Assigned: assigned.Contains(plan.ID)}
		if reason := plan.DeleteBlockReason(assigned.Contains(plan.ID)); reason != "" {
			item.DeleteDisabledReason = &reason
		}
		items = append(items, item)
	}
	respondJSON(w, items)
}

func (s *Server) adminGetOrganizationBillingPlan(w http.ResponseWriter, r *http.Request) {
	if !billingmodels.BillingEnabled() {
		http.Error(w, "Billing is not available", http.StatusNotFound)
		return
	}
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	response, err := describeOrganizationBillingPlanJSON(database.DB(r.Context()), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization billing plan: %v", err)
		http.Error(w, "Failed to load organization billing plan", http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("check_polar") == "true" {
		response.Polar = polar.CheckBillingDiagnostics(r.Context(), orgID, response.PolarCustomerID)
		response.PolarSubscriptionVerification = response.Polar.SubscriptionVerification
	}
	respondJSON(w, response)
}

func (s *Server) adminSetOrganizationBillingPlan(w http.ResponseWriter, r *http.Request) {
	if !billingmodels.BillingEnabled() {
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
	snapshot, err := billingmodels.ResolveOrganizationBillingPlan(db, orgID)
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
		http.Error(w, billingmodels.ErrPolarManagedBillingPlan.Error(), http.StatusBadRequest)
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
		_, err = billingmodels.SetAdminOrganizationPlan(db, orgID, planName, trialEndsAt)
	}
	if errors.Is(err, billingmodels.ErrBillingPlanChanged) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := describeOrganizationBillingPlanJSON(database.DB(r.Context()), orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization billing plan: %v", err)
		http.Error(w, "Failed to load organization billing plan", http.StatusInternalServerError)
		return
	}
	response.PolarSubscriptionVerification = verification
	respondJSON(w, response)
}

func describeOrganizationBillingPlanJSON(tx *gorm.DB, orgID uuid.UUID) (organizationBillingPlanResponse, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return organizationBillingPlanResponse{}, err
	}
	response := organizationBillingPlanResponse{
		Plan: billingmodels.BillingPlanNone, CatalogKey: billingmodels.BillingPlanNone,
		Polar: polar.UncheckedBillingDiagnostics(),
	}
	response.PolarSubscriptionVerification = response.Polar.SubscriptionVerification
	if err := loadStoredBillingEvidence(tx, orgID, &response); err != nil {
		return response, err
	}
	if plan == nil {
		return response, nil
	}

	subscriptionID := ""
	if plan.PolarSubscriptionID != nil {
		subscriptionID = *plan.PolarSubscriptionID
	}
	response.RecordExists = true
	response.UpdatedAt = formatOptionalTime(&plan.UpdatedAt)
	response.TrialStartedAt = formatOptionalTime(plan.TrialStartedAt)
	response.CurrentPeriodStart = formatOptionalTime(plan.CurrentPeriodStart)
	response.PolarSubscriptionID = subscriptionID
	response.PolarSubscriptionURL = polar.SubscriptionDashboardURL(subscriptionID)
	response.CatalogID = plan.PlanID
	response.CatalogKey = plan.Catalog.Key
	response.CatalogName = plan.Catalog.Name
	response.CatalogCategory = plan.Catalog.Category
	response.CreditExpiration = plan.Catalog.CreditExpiration
	response.CreditValidityDays = plan.Catalog.CreditValidityDays
	response.Plan = plan.Plan
	response.PlanSource = plan.PlanSource
	response.PolarSubscriptionStatus = plan.PolarSubscriptionStatus
	response.PolarManaged = billingmodels.OrganizationBillingIsPolarManaged(plan)
	response.TrialEndsAt = formatOptionalTime(plan.TrialEndsAt)
	response.CurrentPeriodEnd = formatOptionalTime(plan.CurrentPeriodEnd)
	return response, nil
}

func trialEndForAdminPlan(tx *gorm.DB, planName string, raw *string) (*time.Time, error) {
	catalog, err := billingmodels.FindBillingPlan(tx, planName)
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

func describeOrganizationLLMCreditJSON(tx *gorm.DB, orgID uuid.UUID) (organizationLLMCreditResponse, error) {
	summary, err := billingmodels.DescribeOrganizationLLMCredit(tx, orgID)
	if err != nil {
		return organizationLLMCreditResponse{}, err
	}
	orgSettings, err := billingmodels.FindOrganizationLLMSettings(tx, orgID)
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
