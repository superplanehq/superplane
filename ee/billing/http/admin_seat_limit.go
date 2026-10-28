package billinghttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func (s *Server) adminOrganizationSeatLimit(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}
	var limit *int
	if r.Method == http.MethodPut {
		var body struct {
			SeatLimit json.RawMessage `json:"seat_limit"`
		}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil || len(body.SeatLimit) == 0 {
			http.Error(w, "seat_limit must be a positive integer or null", http.StatusBadRequest)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if !bytes.Equal(bytes.TrimSpace(body.SeatLimit), []byte("null")) {
			var value int32
			if err := json.Unmarshal(body.SeatLimit, &value); err != nil || value <= 0 {
				http.Error(w, "seat_limit must be a positive integer or null", http.StatusBadRequest)
				return
			}
			capacity := int(value)
			limit = &capacity
		}
	}
	var effective *entitlements.Entitlements
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		switch r.Method {
		case http.MethodPut:
			if err := billingmodels.SetOrganizationSeatLimitOverride(tx, orgID, limit); err != nil {
				return err
			}
		case http.MethodDelete:
			if err := billingmodels.DeleteOrganizationSeatLimitOverride(tx, orgID); err != nil {
				return err
			}
		}
		var err error
		effective, err = entitlements.Resolve(tx, orgID)
		return err
	})
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	respondJSON(w, map[string]any{"plan_limit": effective.PlanSeatLimit, "override": effective.SeatLimitOverride, "effective_limit": effective.SeatLimit})
}
