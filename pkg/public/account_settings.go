package public

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxAccountNameLength = 80

var (
	errAccountNameRequired = errors.New("name is required")
	errAccountNameTooLong  = errors.New("name is too long")
)

type updateAccountRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type deleteAccountRequest struct {
	Email string `json:"email"`
}

func accountProviderResponses(providers []models.AccountProvider) []AccountProviderResponse {
	responses := make([]AccountProviderResponse, 0, len(providers))
	for _, provider := range providers {
		responses = append(responses, AccountProviderResponse{
			Provider: provider.Provider,
			Email:    provider.Email,
			Username: provider.Username,
		})
	}
	return responses
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if hasActiveImpersonation(r) {
		http.Error(w, "Account updates are not allowed while impersonating", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req updateAccountRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == nil && req.Email == nil {
		http.Error(w, "Name or email is required", http.StatusBadRequest)
		return
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return errAccountNameRequired
			}
			if utf8.RuneCountInString(name) > maxAccountNameLength {
				return errAccountNameTooLong
			}
			if err := account.UpdateName(tx, name); err != nil {
				return err
			}
		}
		if req.Email != nil {
			return account.SetEmail(tx, *req.Email)
		}
		return nil
	})
	if errors.Is(err, errAccountNameRequired) {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errAccountNameTooLong) {
		http.Error(w, "Name is too long", http.StatusBadRequest)
		return
	}
	if errors.Is(err, models.ErrAccountEmailNotFromSignInMethod) {
		http.Error(w, "Choose an email from a connected sign-in method.", http.StatusBadRequest)
		return
	}
	if errors.Is(err, models.ErrAccountEmailInUse) {
		http.Error(w, "This email already belongs to another SuperPlane account.", http.StatusConflict)
		return
	}
	if err != nil {
		log.Errorf("Error updating account %s: %v", account.ID, err)
		http.Error(w, "Failed to update account", http.StatusInternalServerError)
		return
	}

	s.getAccount(w, r)
}

func (s *Server) disconnectAccountProvider(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if hasActiveImpersonation(r) {
		http.Error(w, "Sign-in methods cannot change while impersonating", http.StatusForbidden)
		return
	}

	provider := mux.Vars(r)["provider"]
	if provider != models.ProviderGitHub && provider != models.ProviderGoogle {
		http.Error(w, "Unknown sign-in method", http.StatusBadRequest)
		return
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return account.DisconnectProvider(tx, provider)
	})
	if errors.Is(err, models.ErrLastSignInMethod) {
		http.Error(w, "Keep at least one sign-in method.", http.StatusConflict)
		return
	}
	if errors.Is(err, models.ErrSignInMethodNotConnected) {
		http.Error(w, "This sign-in method is not connected.", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Errorf("Error disconnecting %s for account %s: %v", provider, account.ID, err)
		http.Error(w, "Failed to disconnect sign-in method", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if hasActiveImpersonation(r) {
		http.Error(w, "Account deletion is not allowed while impersonating", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req deleteAccountRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if utils.NormalizeEmail(req.Email) != account.Email {
		http.Error(w, "Type your email to confirm account deletion.", http.StatusBadRequest)
		return
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(account, "id = ?", account.ID).Error; err != nil {
			return err
		}
		users, err := models.ListActiveHumanUsersForAccount(tx, account.ID)
		if err != nil {
			return err
		}
		if err := s.refuseAccountDeleteGuards(r, tx, account, users); err != nil {
			return err
		}
		if err := account.SoftDelete(tx, time.Now()); err != nil {
			return err
		}
		return s.removeAccountOrganizationRoles(r.Context(), users)
	})
	if errors.Is(err, models.ErrAccountDeleteLastUncreatedOwner) {
		http.Error(w, "Transfer ownership of organizations you did not create before you delete this account.", http.StatusConflict)
		return
	}
	if errors.Is(err, models.ErrAccountDeleteLastInstallationAdmin) {
		http.Error(w, "Promote another installation admin before you delete this account.", http.StatusConflict)
		return
	}
	if err != nil {
		log.Errorf("Error deleting account %s: %v", account.ID, err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	authentication.ClearAccountCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) refuseAccountDeleteGuards(r *http.Request, tx *gorm.DB, account *models.Account, users []models.User) error {
	if account.IsInstallationAdmin() {
		count, err := models.CountActiveInstallationAdmins(tx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return models.ErrAccountDeleteLastInstallationAdmin
		}
	}

	for _, user := range users {
		organization, err := models.FindOrganizationByIDInTransaction(tx, user.OrganizationID.String())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if organization.CreatedByAccountID != nil && *organization.CreatedByAccountID == account.ID {
			continue
		}

		ownerIDs, err := s.authService.GetOrgUsersForRole(r.Context(), models.RoleOrgOwner, organization.ID.String())
		if err != nil {
			return err
		}
		livingIDs, err := livingOwnerIDs(tx, organization.ID.String(), ownerIDs)
		if err != nil {
			return err
		}
		if len(livingIDs) <= 1 && slices.Contains(livingIDs, user.ID.String()) {
			return models.ErrAccountDeleteLastUncreatedOwner
		}
	}

	return nil
}

func livingOwnerIDs(tx *gorm.DB, organizationID string, ownerIDs []string) ([]string, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	living, err := models.ListActiveUsersByIDInTransaction(tx, organizationID, ownerIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(living))
	for i := range living {
		ids = append(ids, living[i].ID.String())
	}
	return ids, nil
}

func (s *Server) removeAccountOrganizationRoles(ctx context.Context, users []models.User) error {
	for _, user := range users {
		roles, err := s.authService.GetUserRolesForOrg(ctx, user.ID.String(), user.OrganizationID.String())
		if err != nil {
			return err
		}
		for _, role := range roles {
			if err := s.authService.RemoveRole(user.ID.String(), role.Name, user.OrganizationID.String(), models.DomainTypeOrganization); err != nil {
				return err
			}
		}
	}
	return nil
}
