package public

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type SetupOwnerRequest struct {
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Password      string `json:"password"`
	SMTPEnabled   bool   `json:"smtp_enabled"`
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      int    `json:"smtp_port"`
	SMTPUsername  string `json:"smtp_username"`
	SMTPPassword  string `json:"smtp_password"`
	SMTPFromName  string `json:"smtp_from_name"`
	SMTPFromEmail string `json:"smtp_from_email"`
	SMTPUseTLS    bool   `json:"smtp_use_tls"`
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

	if req.SMTPEnabled {
		if req.SMTPHost == "" || req.SMTPPort == 0 || req.SMTPFromEmail == "" {
			http.Error(w, "SMTP host, port, and from email are required", http.StatusBadRequest)
			return
		}
		if req.SMTPUsername != "" && req.SMTPPassword == "" {
			http.Error(w, "SMTP password is required when username is provided", http.StatusBadRequest)
			return
		}
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

		if req.SMTPEnabled {
			var encryptedPassword []byte
			if req.SMTPPassword != "" {
				encryptedPassword, err = s.encryptor.Encrypt(
					context.Background(),
					[]byte(req.SMTPPassword),
					[]byte("smtp_password"),
				)
				if err != nil {
					return err
				}
			}

			err = models.UpsertEmailSettingsInTransaction(tx, &models.EmailSettings{
				Provider:      models.EmailProviderSMTP,
				SMTPHost:      req.SMTPHost,
				SMTPPort:      req.SMTPPort,
				SMTPUsername:  req.SMTPUsername,
				SMTPPassword:  encryptedPassword,
				SMTPFromName:  req.SMTPFromName,
				SMTPFromEmail: req.SMTPFromEmail,
				SMTPUseTLS:    req.SMTPUseTLS,
			})
			if err != nil {
				return err
			}
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
