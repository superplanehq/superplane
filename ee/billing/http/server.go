package billinghttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

// Server owns the billing HTTP handlers. Core owns the routers and middleware.
type Server struct {
	encryptor        crypto.Encryptor
	registry         *registry.Registry
	installationURLs func() (baseURL, webhooksBaseURL string)
}

var _ billing.HTTPRegistrar = (*Server)(nil)

func New(encryptor crypto.Encryptor, registry *registry.Registry, installationURLs func() (string, string)) *Server {
	return &Server{encryptor: encryptor, registry: registry, installationURLs: installationURLs}
}

func (s *Server) RegisterPublic(publicRoute *mux.Router) {
	publicRoute.HandleFunc("/api/v1/polar/webhooks", s.handlePolarWebhook).Methods("POST")
}

func (s *Server) RegisterAdmin(adminRoute *mux.Router) {
	adminRoute.HandleFunc("/stats", s.adminRegistrationStats).Methods("GET")
	adminRoute.HandleFunc("/billing-plans", s.adminListBillingPlans).Methods("GET")
	adminRoute.HandleFunc("/credit-packs", s.adminListCreditPacks).Methods("GET")
	adminRoute.HandleFunc("/credit-packs", s.adminSaveCreditPack).Methods("POST")
	adminRoute.HandleFunc("/credit-packs/{packId}", s.adminSaveCreditPack).Methods("PUT")
	adminRoute.HandleFunc("/credit-packs/{packId}", s.adminDeleteCreditPack).Methods("DELETE")
	adminRoute.HandleFunc("/polar/products", s.adminListPolarProducts).Methods("GET")
	adminRoute.HandleFunc("/organizations/{orgId}/llm-credit", s.adminGetOrganizationLLMCredit).Methods("GET")
	adminRoute.HandleFunc("/organizations/{orgId}/llm-credit/grants", s.adminAddOrganizationLLMCredit).Methods("POST")
	adminRoute.HandleFunc("/organizations/{orgId}/llm-credit/grants", s.adminListOrganizationCreditGrants).Methods("GET")
	adminRoute.HandleFunc("/organizations/{orgId}/llm-credit/balances", s.adminSetOrganizationCreditBalance).Methods("PUT")
	adminRoute.HandleFunc("/organizations/{orgId}/llm-settings", s.adminUpdateOrganizationLLMMarkup).Methods("PATCH")
	adminRoute.HandleFunc("/billing-plans", s.adminSaveBillingPlan).Methods("POST")
	adminRoute.HandleFunc("/billing-plans/{planId}", s.adminSaveBillingPlan).Methods("PUT")
	adminRoute.HandleFunc("/billing-plans/{planId}", s.adminDeleteBillingPlan).Methods("DELETE")
	adminRoute.HandleFunc("/feature-definitions/{featureKey}/organizations", s.adminListFeatureOrganizations).Methods("GET")
	adminRoute.HandleFunc("/feature-definitions", s.adminListFeatureDefinitions).Methods("GET")
	adminRoute.HandleFunc("/feature-definitions", s.adminSaveFeatureDefinition).Methods("POST")
	adminRoute.HandleFunc("/feature-definitions/{featureKey}", s.adminSaveFeatureDefinition).Methods("PUT")
	adminRoute.HandleFunc("/feature-definitions/{featureKey}", s.adminDeleteFeatureDefinition).Methods("DELETE")
	adminRoute.HandleFunc("/organizations/{orgId}/seat-limit", s.adminOrganizationSeatLimit).Methods("GET", "PUT", "DELETE")
	adminRoute.HandleFunc("/organizations/{orgId}/features", s.adminOrganizationFeatures).Methods("GET", "PUT")
	adminRoute.HandleFunc("/organizations/{orgId}/billing-plan", s.adminGetOrganizationBillingPlan).Methods("GET")
	adminRoute.HandleFunc("/organizations/{orgId}/billing-customer/sync", s.adminSyncOrganizationBillingCustomer).Methods("POST")
	adminRoute.HandleFunc("/organizations/{orgId}/billing-plan", s.adminSetOrganizationBillingPlan).Methods("PUT")
	adminRoute.HandleFunc("/organizations/{orgId}/spending-report", s.adminGetOrganizationSpendingReport).Methods("GET")
	adminRoute.HandleFunc("/polar/webhooks", s.adminListPolarWebhooks).Methods("GET")
	adminRoute.HandleFunc("/polar/webhooks/endpoints", s.adminListPolarWebhookEndpoints).Methods("GET")
	adminRoute.HandleFunc("/polar/webhooks/{eventId}/redeliver", s.adminRedeliverPolarWebhook).Methods("POST")
	adminRoute.HandleFunc("/price-books", s.adminGetPriceBooks).Methods("GET")
	adminRoute.HandleFunc("/price-books", s.adminSavePriceBooks).Methods("PUT")
	adminRoute.HandleFunc("/price-books", s.adminDeletePriceBook).Methods("DELETE")
	adminRoute.HandleFunc("/price-books/sync", s.adminSyncPriceBooks).Methods("POST")
	adminRoute.HandleFunc("/price-books/current", s.adminActivatePriceBook).Methods("PUT")
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

type paginatedResponse struct {
	Items  any   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

func parsePagination(r *http.Request) (search string, limit, offset int) {
	search = r.URL.Query().Get("search")
	limit = 50
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > 200 {
		limit = 200
	}
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	return search, limit, offset
}
