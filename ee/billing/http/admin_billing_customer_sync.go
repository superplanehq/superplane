package billinghttp

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type organizationBillingCustomerSyncResponse struct {
	BillingPlan organizationBillingPlanResponse `json:"billing_plan"`
	Error       string                          `json:"error,omitempty"`
}

// adminSyncOrganizationBillingCustomer repairs only the local customer mapping.
// Card availability remains a live Polar check, never a stored override.
func (s *Server) adminSyncOrganizationBillingCustomer(w http.ResponseWriter, r *http.Request) {
	if !billingmodels.BillingEnabled() {
		http.Error(w, "Billing is not available", http.StatusNotFound)
		return
	}
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	db := database.DB(r.Context())
	response, err := describeOrganizationBillingPlanJSON(db, orgID)
	if err != nil {
		log.WithError(err).Error("admin: failed to load billing diagnostics for customer sync")
		http.Error(w, "Could not load billing diagnostics. Try again.", http.StatusInternalServerError)
		return
	}
	// Never trust the customer's ID or card status from an earlier browser check.
	// Network requests complete before taking the organization lock.
	diagnostics := polar.CheckBillingDiagnostics(r.Context(), orgID, response.PolarCustomerID)
	response.Polar = diagnostics
	response.PolarSubscriptionVerification = diagnostics.SubscriptionVerification
	if status, message := billingCustomerSyncFailure(diagnostics); message != "" {
		respondBillingCustomerSync(w, response, status, message)
		return
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		return billingmodels.SetOrganizationPolarCustomerID(tx, orgID, diagnostics.CustomerID)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondBillingCustomerSync(w, response, http.StatusNotFound, "Organization not found.")
			return
		}
		log.WithError(err).Error("admin: failed to sync Polar customer")
		respondBillingCustomerSync(w, response, http.StatusInternalServerError, "Could not save the Polar customer ID. Try again.")
		return
	}
	response, err = describeOrganizationBillingPlanJSON(db, orgID)
	if err != nil {
		log.WithError(err).Error("admin: failed to load billing diagnostics after customer sync")
		http.Error(w, "Customer ID saved. Could not reload billing diagnostics. Refresh and try again.", http.StatusInternalServerError)
		return
	}
	diagnostics.CustomerIDMismatch = diagnostics.CustomerID != response.PolarCustomerID
	response.Polar = diagnostics
	response.PolarSubscriptionVerification = diagnostics.SubscriptionVerification
	respondBillingCustomerSync(w, response, http.StatusOK, "")
}

func billingCustomerSyncFailure(diagnostics polar.BillingDiagnostics) (int, string) {
	if !diagnostics.Configured {
		return http.StatusServiceUnavailable, "Polar is not configured. Configure Polar before syncing the customer."
	}
	if diagnostics.CustomerStatus == "absent" {
		return http.StatusUnprocessableEntity, "No Polar customer was found for this organization."
	}
	if diagnostics.CustomerStatus != "found" {
		return http.StatusBadGateway, "Could not verify the Polar customer. Check Polar and try again."
	}
	if diagnostics.CardStatus == "absent" {
		return http.StatusUnprocessableEntity, "No usable saved card was found in Polar. Add or update the card in Polar, then try again."
	}
	if diagnostics.CardStatus != "usable" {
		return http.StatusBadGateway, "Could not verify the saved card in Polar. Try again."
	}
	return http.StatusOK, ""
}

func respondBillingCustomerSync(w http.ResponseWriter, response organizationBillingPlanResponse, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	respondJSON(w, organizationBillingCustomerSyncResponse{BillingPlan: response, Error: message})
}
