package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
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
	if hasActiveSupportSession(r) {
		http.Error(w, "Account updates are not allowed in a read-only support session", http.StatusForbidden)
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
	if hasActiveSupportSession(r) {
		http.Error(w, "Sign-in methods cannot change in a read-only support session", http.StatusForbidden)
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
	if hasActiveSupportSession(r) {
		http.Error(w, "Account deletion is not allowed in a read-only support session", http.StatusForbidden)
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

	_, err = s.softDeleteAccount(r.Context(), account)
	if errors.Is(err, billing.ErrOrganizationSubscriptionLive) {
		log.WithError(err).WithField("account_id", account.ID.String()).Info("account deletion blocked by a live subscription")
		http.Error(w, billing.ErrOrganizationSubscriptionLive.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, billing.ErrOrganizationCheckoutPending) {
		log.WithError(err).WithField("account_id", account.ID.String()).Info("account deletion blocked by a pending checkout")
		http.Error(w, billing.ErrOrganizationCheckoutPending.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, billing.ErrOrganizationSubscriptionUnverified) {
		log.WithError(err).WithField("account_id", account.ID.String()).Warn("account deletion could not verify subscription termination")
		http.Error(w, billing.ErrOrganizationSubscriptionUnverified.Error(), http.StatusServiceUnavailable)
		return
	}
	if errors.Is(err, billing.ErrBillingPlanChanged) {
		log.WithField("account_id", account.ID.String()).Info("account deletion stopped by a billing change")
		http.Error(w, "The billing record changed during deletion. Try again.", http.StatusConflict)
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

// softDeleteAccount deletes the account and every organization where it is
// the only owner. It returns the organizations that were deleted.
//
// The account row lock comes first, before any organization lock. Invitation
// acceptance takes the locks in the same order, so the two never deadlock and
// one of them always sees the result of the other. Membership, grants, and
// capacity change in one transaction, and no Polar call holds a lock.
func (s *Server) softDeleteAccount(ctx context.Context, account *models.Account) ([]models.Organization, error) {
	// Ask Polar before the transaction: the requests must not hold the
	// organization locks. An organization that becomes sole-owner after this
	// point has no clearance, so the deletion stops.
	clearances, err := verifyAccountOrganizationDeletions(ctx, account)
	if err != nil {
		return nil, err
	}

	var deletedOrganizations []models.Organization
	var capacityAlerts billing.CapacityAlerts
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockAccount(tx, account.ID)
		if err != nil {
			return err
		}
		*account = *locked
		// A membership that an acceptance committed before this lock is in
		// the list, so the deletion removes it too.
		users, err := models.ListActiveHumanUsersForAccount(tx, account.ID)
		if err != nil {
			return err
		}
		if err := s.refuseAccountDeleteGuards(tx, account); err != nil {
			return err
		}
		deletedOrganizations, err = models.ListOrganizationsPendingAccountDeletion(tx, account.ID)
		if err != nil {
			return err
		}
		// SoftDelete locks all affected organizations in ID order. Keep
		// those locks before removing grants, as member removal does.
		if err := account.SoftDelete(tx, time.Now(), clearances); err != nil {
			return err
		}
		if err := s.removeAccountOrganizationGrants(tx, users); err != nil {
			return err
		}
		capacityAlerts, err = recheckSeatCapacityAfterAccountDeletion(tx, survivingOrganizationIDs(users, deletedOrganizations))
		return err
	})
	if err != nil {
		return nil, err
	}
	// The alerts describe committed state only. Each surviving organization
	// keeps its paid seats: account deletion never changes a Polar seat count.
	capacityAlerts.Emit()
	return deletedOrganizations, nil
}

func verifyAccountOrganizationDeletions(ctx context.Context, account *models.Account) (map[uuid.UUID]billing.DeletionGuard, error) {
	pending, err := models.ListOrganizationsPendingAccountDeletion(database.DB(ctx), account.ID)
	if err != nil {
		return nil, err
	}
	clearances := make(map[uuid.UUID]billing.DeletionGuard, len(pending))
	for _, organization := range pending {
		clearance, err := billing.PrepareDeletion(ctx, organization.ID)
		if err != nil {
			return nil, fmt.Errorf("organization %s: %w", organization.ID, err)
		}
		clearances[organization.ID] = clearance
	}
	return clearances, nil
}

// recheckSeatCapacityAfterAccountDeletion updates the over-capacity record of
// each organization that keeps running after the account left it.
func recheckSeatCapacityAfterAccountDeletion(tx *gorm.DB, organizationIDs []uuid.UUID) (billing.CapacityAlerts, error) {
	checks := make(billing.CapacityAlerts, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		check, err := billing.RecheckSeatCapacity(tx, organizationID, "account_deletion")
		if err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, nil
}

// survivingOrganizationIDs returns, once each, the organizations where the
// account had a human membership and that the deletion does not delete.
func survivingOrganizationIDs(users []models.User, deletedOrganizations []models.Organization) []uuid.UUID {
	var organizationIDs []uuid.UUID
	for _, user := range users {
		if slices.Contains(organizationIDs, user.OrganizationID) {
			continue
		}
		isDeleted := slices.ContainsFunc(deletedOrganizations, func(organization models.Organization) bool {
			return organization.ID == user.OrganizationID
		})
		if isDeleted {
			continue
		}
		organizationIDs = append(organizationIDs, user.OrganizationID)
	}
	return organizationIDs
}

func (s *Server) refuseAccountDeleteGuards(tx *gorm.DB, account *models.Account) error {
	if !account.IsInstallationAdmin() {
		return nil
	}

	count, err := models.CountActiveInstallationAdmins(tx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return models.ErrAccountDeleteLastInstallationAdmin
	}
	return nil
}

// removeAccountOrganizationGrants removes the direct roles and the group
// memberships of each membership in the deletion transaction, so a deleted
// account keeps no organization authority.
func (s *Server) removeAccountOrganizationGrants(tx *gorm.DB, users []models.User) error {
	for _, user := range users {
		if err := s.authService.RemoveUserFromOrganizationInTransaction(tx, user.ID.String(), user.OrganizationID.String()); err != nil {
			return fmt.Errorf("remove organization grants of user %s: %w", user.ID, err)
		}
	}
	return nil
}
