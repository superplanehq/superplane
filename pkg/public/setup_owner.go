package public

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
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
	OrganizationID string `json:"organization_id"`
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

		organizationName := req.FirstName + "'s organization"
		organization, err = models.CreateOrganizationInTransaction(tx, organizationName, "")
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

		return nil
	})

	if err != nil {
		log.Errorf("Failed to set up owner account: %v", err)
		http.Error(w, "Failed to set up owner account", http.StatusInternalServerError)
		return
	}

	// Mark setup as completed so we stop redirecting to /setup
	middleware.MarkOwnerSetupCompleted()

	// Create account cookie so the owner is signed in
	token, err := s.jwt.Generate(account.ID.String(), 24*time.Hour)
	if err != nil {
		log.Errorf("Failed to generate account token for owner: %v", err)
		http.Error(w, "Failed to create owner session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(24 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SetupOwnerResponse{
		OrganizationID: organization.ID.String(),
	})
}

