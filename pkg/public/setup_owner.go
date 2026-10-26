package public

import (
	"encoding/json"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

const (
	setupOwnerStatusDisabled   = "disabled"
	setupOwnerStatusRequired   = "required"
	setupOwnerStatusInProgress = "in_progress"
	setupOwnerStatusComplete   = "complete"
)

type SetupOwnerRequest struct {
	Email              string `json:"email"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Password           string `json:"password"`
	ShareOwnerIdentity bool   `json:"share_owner_identity"`
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

	fullName := req.FirstName + " " + req.LastName

	var organization *models.Organization
	var account *models.Account
	var user *models.User

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error

		account, err = models.CreateAccountInTransaction(tx, fullName, req.Email)
		if err != nil {
			return err
		}

		// The first account is automatically an installation admin
		if err := tx.Model(account).Update("installation_admin", true).Error; err != nil {
			return err
		}
		account.InstallationAdmin = true

		// Hash and store password
		passwordHash, err := crypto.HashPassword(req.Password)
		if err != nil {
			return err
		}

		_, err = models.CreateAccountPasswordAuthInTransaction(tx, account.ID, passwordHash)
		if err != nil {
			return err
		}

		organization, err = models.CreateOrganizationInTransaction(tx, "Demo", "")
		if err != nil {
			return err
		}

		user, err = models.CreateUserInTransaction(tx, organization.ID, account.ID, req.Email, fullName)
		if err != nil {
			return err
		}

		err = s.authService.SetupOrganization(tx, organization.ID.String(), user.ID.String())
		if err != nil {
			return err
		}

		if err := models.SetOrganizationCreatedByAccount(tx, organization.ID, account.ID); err != nil {
			return err
		}
		if billing.Enabled() {
			if err := billing.GrantWelcomeCredit(tx, organization.ID, account.ID); err != nil {
				return err
			}
		}

		if err := models.SetShareOwnerIdentity(tx, req.ShareOwnerIdentity); err != nil {
			return err
		}

		// Account creation is not the end of the wizard: the owner still
		// goes through license, SMTP, and fleet steps. Record the next step
		// so a refresh resumes the wizard instead of reopening the form.
		return models.SetOwnerSetupStep(tx, models.OwnerSetupStepLicense)
	})

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

type SetupOwnerStatusResponse struct {
	Status           string `json:"status"`
	Step             string `json:"step,omitempty"`
	BeaconEnabled    bool   `json:"beacon_enabled"`
	InstallationType string `json:"installation_type"`
}

func setupOwnerStatusResponse(status, step string) SetupOwnerStatusResponse {
	return SetupOwnerStatusResponse{
		Status:           status,
		Step:             step,
		BeaconEnabled:    telemetry.BeaconEnabled(),
		InstallationType: os.Getenv("SUPERPLANE_INSTALLATION_TYPE"),
	}
}

// setupOwnerStatus reports the owner setup wizard state to any caller.
// It must not expose organization data: an anonymous caller only learns
// whether the wizard is available, required, in progress, or closed.
func (s *Server) setupOwnerStatus(w http.ResponseWriter, r *http.Request) {
	if !middleware.OwnerSetupEnabled() {
		respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusDisabled, ""))
		return
	}

	if middleware.IsOwnerSetupRequired() {
		respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusRequired, ""))
		return
	}

	step, err := models.GetOwnerSetupStep(database.DB(r.Context()))
	if err != nil {
		log.Errorf("Failed to read owner setup step: %v", err)
		http.Error(w, "Failed to read owner setup status", http.StatusInternalServerError)
		return
	}

	if step == "" {
		respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusComplete, ""))
		return
	}

	respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusInProgress, step))
}

type SetupOwnerProgressRequest struct {
	Step string `json:"step"`
}

// updateSetupOwnerProgress moves the owner setup wizard to a new step, or
// closes it when the step is cleared. Only an installation admin can write
// progress, and a closed wizard cannot be reopened.
func (s *Server) updateSetupOwnerProgress(w http.ResponseWriter, r *http.Request) {
	if !middleware.OwnerSetupEnabled() {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !account.IsInstallationAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req SetupOwnerProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Step != "" && !models.IsOwnerSetupStep(req.Step) {
		http.Error(w, "Unknown setup step", http.StatusBadRequest)
		return
	}

	var updated bool
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var err error
		updated, err = models.SetOwnerSetupStepIfOpen(tx, req.Step)
		return err
	})
	if err != nil {
		log.Errorf("Failed to update owner setup step: %v", err)
		http.Error(w, "Failed to update owner setup progress", http.StatusInternalServerError)
		return
	}

	if !updated {
		http.Error(w, "Owner setup is already finished", http.StatusConflict)
		return
	}

	if req.Step == "" {
		respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusComplete, ""))
		return
	}

	respondJSON(w, setupOwnerStatusResponse(setupOwnerStatusInProgress, req.Step))
}
