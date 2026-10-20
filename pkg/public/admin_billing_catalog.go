package public

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"net/http"
	"slices"
)

func (s *Server) adminListBillingPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := models.ListBillingPlans(database.DB(r.Context()))
	if err != nil {
		http.Error(w, "Failed to load billing plans", 500)
		return
	}
	respondJSON(w, plans)
}

func (s *Server) adminSaveBillingPlan(w http.ResponseWriter, r *http.Request) {
	var plan models.BillingPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		http.Error(w, "Invalid billing plan", 400)
		return
	}
	id := mux.Vars(r)["planId"]
	plan.ID = uuid.Nil
	if id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid plan ID", 400)
			return
		}
		plan.ID = parsed
	}
	for _, key := range plan.Features {
		if !slices.ContainsFunc(entitlements.Definitions, func(feature entitlements.Feature) bool { return feature.Key == key }) {
			http.Error(w, "Unknown feature", 400)
			return
		}
	}
	if err := models.SaveBillingPlan(database.DB(r.Context()), &plan); err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, plan)
}

func (s *Server) adminDeleteBillingPlan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["planId"])
	if err != nil {
		http.Error(w, "Invalid plan ID", 400)
		return
	}
	if err := models.DeleteBillingPlan(database.DB(r.Context()), id); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminListFeatureDefinitions(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, entitlements.Definitions)
}

func (s *Server) adminOrganizationFeatures(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	db := database.DB(r.Context())
	if r.Method == http.MethodPut {
		var grant models.OrganizationFeatureGrant
		if err := json.NewDecoder(r.Body).Decode(&grant); err != nil {
			http.Error(w, "Invalid feature grant", 400)
			return
		}
		if !slices.ContainsFunc(entitlements.Definitions, func(feature entitlements.Feature) bool { return feature.Key == grant.FeatureKey }) {
			http.Error(w, "Unknown feature", 400)
			return
		}
		grant.OrganizationID = orgID
		if err := models.SetOrganizationFeatureGrant(db, grant); err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	effective, err := entitlements.Resolve(db, orgID)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	grants, err := models.ListOrganizationFeatureGrants(db, orgID)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, map[string]any{"grants": grants, "effective": effective})
}

func writeCatalogError(w http.ResponseWriter, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Billing plan not found", 404)
		return
	}
	if errors.Is(err, models.ErrCatalogPlan) {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Error(w, "Failed to update billing catalog", 500)
}
