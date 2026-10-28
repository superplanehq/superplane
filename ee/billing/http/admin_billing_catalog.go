package billinghttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

const (
	billingPlanUpdateErrorMessage = "Could not update billing plans. Try again."
	billingPlanLoadErrorMessage   = "Could not load billing plans. Try again."
)

// billingPlanErrorResponse is the JSON body for billing-plan route failures.
type billingPlanErrorResponse struct {
	Message string `json:"message"`
}

func (s *Server) adminSaveBillingPlan(w http.ResponseWriter, r *http.Request) {
	var plan billingmodels.BillingPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeBillingPlanJSONError(w, http.StatusBadRequest, "Invalid billing plan.")
		return
	}
	id := mux.Vars(r)["planId"]
	plan.ID = uuid.Nil
	if id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			writeBillingPlanJSONError(w, http.StatusBadRequest, "Invalid billing plan ID.")
			return
		}
		plan.ID = parsed
	}
	if status, err := validateSeatPlanProduct(r.Context(), &plan); err != nil {
		writeBillingPlanJSONError(w, status, err.Error())
		return
	}
	if err := billingmodels.SaveBillingPlan(database.DB(r.Context()), &plan); err != nil {
		writeBillingPlanError(w, r, err)
		return
	}
	respondJSON(w, plan)
}

func (s *Server) adminDeleteBillingPlan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["planId"])
	if err != nil {
		writeBillingPlanJSONError(w, http.StatusBadRequest, "Invalid billing plan ID.")
		return
	}
	if err := billingmodels.DeleteBillingPlan(database.DB(r.Context()), id); err != nil {
		writeBillingPlanError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeBillingPlanError maps a model error from a billing-plan write to a JSON
// response. Unexpected errors keep their cause for error reports only.
func writeBillingPlanError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeBillingPlanJSONError(w, http.StatusNotFound, "Billing plan not found.")
	case errors.Is(err, billingmodels.ErrCatalogPlan):
		writeBillingPlanJSONError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), billingmodels.ErrCatalogPlan.Error()+": "))
	default:
		writeBillingPlanServerError(w, r, err, billingPlanUpdateErrorMessage)
	}
}

// writeBillingPlanServerError reports the cause and returns a safe JSON message.
func writeBillingPlanServerError(w http.ResponseWriter, r *http.Request, err error, message string) {
	middleware.SetServerError(r.Context(), err, nil)
	writeBillingPlanJSONError(w, http.StatusInternalServerError, message)
}

func writeBillingPlanJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(billingPlanErrorResponse{Message: message})
}

func (s *Server) adminListFeatureDefinitions(w http.ResponseWriter, r *http.Request) {
	definitions, err := billingmodels.ListPlanFeatureDefinitions(database.DB(r.Context()))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, definitions)
}

func (s *Server) adminSaveFeatureDefinition(w http.ResponseWriter, r *http.Request) {
	feature := billingmodels.PlanFeatureDefinition{Gated: true}
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, "Invalid feature definition", http.StatusBadRequest)
		return
	}
	if key := mux.Vars(r)["featureKey"]; key != "" && feature.Key != key {
		http.Error(w, "Feature keys cannot change", http.StatusBadRequest)
		return
	}
	save := billingmodels.UpdatePlanFeatureDefinition
	if r.Method == http.MethodPost {
		save = billingmodels.CreatePlanFeatureDefinition
	}
	if err := save(database.DB(r.Context()), &feature); err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, feature)
}

func (s *Server) adminDeleteFeatureDefinition(w http.ResponseWriter, r *http.Request) {
	if err := billingmodels.DeletePlanFeatureDefinition(database.DB(r.Context()), mux.Vars(r)["featureKey"]); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminOrganizationFeatures(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	db := database.DB(r.Context())
	if r.Method == http.MethodPut {
		var grant billingmodels.OrganizationFeatureGrant
		if err := json.NewDecoder(r.Body).Decode(&grant); err != nil {
			http.Error(w, "Invalid feature grant", 400)
			return
		}
		grant.OrganizationID = orgID
		if err := billingmodels.SetOrganizationFeatureGrant(db, grant); err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	effective, err := entitlements.Resolve(db, orgID)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	grants, err := billingmodels.ListOrganizationFeatureGrants(db, orgID)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, map[string]any{"grants": grants, "effective": effective})
}

func writeCatalogError(w http.ResponseWriter, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Billing catalog entry not found", 404)
		return
	}
	if errors.Is(err, billingmodels.ErrCatalogPlan) {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Error(w, "Failed to update billing catalog", 500)
}

func (s *Server) adminListFeatureOrganizations(w http.ResponseWriter, r *http.Request) {
	search, limit, offset := parsePagination(r)
	items, total, err := billingmodels.ListFeatureOrganizations(database.DB(r.Context()), mux.Vars(r)["featureKey"], search, limit, offset)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Feature not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load feature organizations", http.StatusInternalServerError)
		return
	}
	respondJSON(w, paginatedResponse{Items: items, Total: total, Limit: limit, Offset: offset})
}
