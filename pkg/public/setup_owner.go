package public

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type SetupOwnerRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
}

type SetupOwnerResponse struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationSlug string `json:"organization_slug"`
}

func (s *Server) setupOwner(w http.ResponseWriter, r *http.Request) {
	if !middleware.OwnerSetupEnabled() {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if !middleware.IsOwnerSetupRequired() {
		http.Error(w, "Instance already initialized", http.StatusConflict)
		return
	}

	var req SetupOwnerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.FirstName == "" || req.LastName == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	var organization *models.Organization
	var account *models.Account

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		organization, account, _, err = createOwnerAccountInTransaction(
			tx,
			s.authService,
			req.Email,
			req.FirstName,
			req.LastName,
			req.Password,
			config.DefaultOwnerBootstrapOrgName,
		)
		return err
	})

	if errors.Is(err, errOwnerSetupAlreadyCompleted) {
		// Another API pod (or the .env bootstrap) finished setup while this
		// request waited on the setup lock. Sync the cached state so this
		// pod stops redirecting requests to /setup.
		middleware.MarkOwnerSetupCompleted()
		http.Error(w, "Instance already initialized", http.StatusConflict)
		return
	}

	if err != nil {
		log.Errorf("Failed to set up owner account: %v", err)
		http.Error(w, "Failed to set up owner account", http.StatusInternalServerError)
		return
	}

	// Mark setup as completed so we stop redirecting to /setup
	middleware.MarkOwnerSetupCompleted()
	s.registry.HTTPContext().InvalidatePolicyCache()

	if err := authentication.IssueAccountSession(w, r, s.jwt, account.ID.String()); err != nil {
		log.Errorf("Failed to generate account token for owner: %v", err)
		http.Error(w, "Failed to create owner session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SetupOwnerResponse{
		OrganizationID:   organization.ID.String(),
		OrganizationSlug: organization.Slug,
	})
}
