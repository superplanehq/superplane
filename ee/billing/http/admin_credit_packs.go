package billinghttp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func (s *Server) adminListCreditPacks(w http.ResponseWriter, r *http.Request) {
	packs, err := billingmodels.ListCreditPacks(database.DB(r.Context()))
	if err != nil {
		writeCreditPackError(w, err)
		return
	}
	respondJSON(w, packs)
}

func (s *Server) adminSaveCreditPack(w http.ResponseWriter, r *http.Request) {
	// Decode only editable fields. Timestamps and deletion state belong to the server.
	var input struct {
		Name           string  `json:"name"`
		AmountCents    int64   `json:"amount_cents"`
		Enabled        bool    `json:"enabled"`
		PolarProductID *string `json:"polar_product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid credit pack.", http.StatusBadRequest)
		return
	}
	pack := billingmodels.CreditPack{Name: input.Name, AmountCents: input.AmountCents, Enabled: input.Enabled, PolarProductID: input.PolarProductID}
	if id := mux.Vars(r)["packId"]; id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid credit pack ID.", http.StatusBadRequest)
			return
		}
		pack.ID = parsed
	}
	if err := billingmodels.SaveCreditPack(database.DB(r.Context()), &pack); err != nil {
		writeCreditPackError(w, err)
		return
	}
	respondJSON(w, pack)
}

func (s *Server) adminDeleteCreditPack(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["packId"])
	if err != nil {
		http.Error(w, "Invalid credit pack ID.", http.StatusBadRequest)
		return
	}
	if err := billingmodels.DeleteCreditPack(database.DB(r.Context()), id); err != nil {
		writeCreditPackError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeCreditPackError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, "Credit pack not found.", http.StatusNotFound)
	case errors.Is(err, billingmodels.ErrCreditPack):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "Could not update credit packs. Try again.", http.StatusInternalServerError)
	}
}
