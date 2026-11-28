package core

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// EmailDomain returns the normalized domain of an email address.
func EmailDomain(email string) (string, bool) {
	normalized := utils.NormalizeEmail(email)
	at := strings.LastIndex(normalized, "@")
	if at <= 0 || at == len(normalized)-1 {
		return "", false
	}
	return normalized[at+1:], true
}

func organizationOwnsVerifiedDomain(tx *gorm.DB, orgID uuid.UUID, domain string) (bool, error) {
	verified, err := models.FindVerifiedOrganizationDomain(tx, domain)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return verified.OrganizationID == orgID, nil
}

// LinkAccount applies the identity linking rules inside the caller's transaction.
// The caller must not hold an organization lock yet. This function locks the
// account before the organization.
func (p *Pipeline) LinkAccount(tx *gorm.DB, conn *models.SSOConnection, external ExternalIdentity, provisionedBy string) (*models.Account, *models.SSOIdentity, error) {
	email := utils.NormalizeEmail(external.Email)
	if email == "" || external.Subject == "" {
		return nil, nil, Fail(CodeFailed)
	}

	identity, err := models.FindSSOIdentityBySubject(tx, conn.ID, external.Subject)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	if identity != nil {
		return p.bindExistingIdentity(tx, conn, identity, external, false)
	}

	pending, err := models.FindSSOIdentityByPendingEmail(tx, conn.ID, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	if pending != nil {
		return p.bindExistingIdentity(tx, conn, pending, external, true)
	}

	account, err := p.AccountForEmail(tx, conn, email, external.Name)
	if err != nil {
		return nil, nil, err
	}

	subject := external.Subject
	identity = &models.SSOIdentity{
		ID:             uuid.New(),
		ConnectionID:   conn.ID,
		OrganizationID: conn.OrganizationID,
		AccountID:      account.ID,
		Subject:        &subject,
		Email:          email,
		Name:           external.Name,
		Active:         true,
		ProvisionedBy:  provisionedBy,
		CreatedAt:      p.now(),
		UpdatedAt:      p.now(),
	}
	return account, identity, nil
}

func (p *Pipeline) bindExistingIdentity(tx *gorm.DB, conn *models.SSOConnection, identity *models.SSOIdentity, external ExternalIdentity, assignSubject bool) (*models.Account, *models.SSOIdentity, error) {
	if identity.OrganizationID != conn.OrganizationID {
		return nil, nil, Fail(CodeFailed)
	}
	if p.BeforeIdentityLock != nil {
		p.BeforeIdentityLock(tx, identity.ID)
	}
	account, err := models.LockAccount(tx, identity.AccountID)
	if err != nil {
		return nil, nil, err
	}
	if account.IsBlocked() {
		return nil, nil, Fail(CodeAccountBlocked)
	}
	locked, err := models.LockSSOIdentity(tx, identity.ConnectionID, identity.ID)
	if err != nil {
		return nil, nil, err
	}
	if locked.OrganizationID != conn.OrganizationID {
		return nil, nil, Fail(CodeFailed)
	}
	if !locked.Active {
		return nil, nil, Fail(CodeAccessDenied)
	}
	// A SCIM profile change does not transfer ownership of the linked account.
	if assignSubject && utils.NormalizeEmail(account.Email) != utils.NormalizeEmail(external.Email) {
		return nil, nil, Fail(CodeAccountConflict)
	}
	locked.Email = utils.NormalizeEmail(external.Email)
	locked.Name = external.Name
	if assignSubject {
		subject := external.Subject
		locked.Subject = &subject
	}
	return account, locked, nil
}

// AccountForEmail applies linking rules 3 and 4. The domain must be verified
// by this organization before an account is linked or created.
func (p *Pipeline) AccountForEmail(tx *gorm.DB, conn *models.SSOConnection, email, name string) (*models.Account, error) {
	domain, ok := EmailDomain(email)
	if !ok {
		return nil, Fail(CodeFailed)
	}
	owned, err := organizationOwnsVerifiedDomain(tx, conn.OrganizationID, domain)
	if err != nil {
		return nil, err
	}
	account, err := models.LockAccountByEmail(tx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if account != nil && account.IsBlocked() {
		return nil, Fail(CodeAccountBlocked)
	}
	if !owned {
		return nil, Fail(CodeAccountConflict)
	}
	if account != nil {
		return account, nil
	}
	created, err := models.CreateAccountInTransaction(tx, name, utils.NormalizeEmail(email))
	if err != nil {
		return nil, err
	}
	return models.LockAccount(tx, created.ID)
}

// EnsureMembership admits the account into the organization when the connection allows it.
// adoptProvisionedBy is set on an existing active member. Sign-in leaves that field empty.
func (p *Pipeline) EnsureMembership(ctx context.Context, tx *gorm.DB, conn *models.SSOConnection, account *models.Account, identity *models.SSOIdentity, provisionedBy, adoptProvisionedBy string, requireJIT bool) error {
	if _, err := models.LockOrganization(tx, conn.OrganizationID); err != nil {
		return err
	}

	user, err := models.FindMaybeDeletedUserByEmailInTransaction(tx, conn.OrganizationID.String(), account.Email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = nil
	} else if err != nil {
		return err
	}

	active := err == nil && user != nil && !user.DeletedAt.Valid
	if active {
		if user.AccountID == nil || *user.AccountID != account.ID {
			return Fail(CodeAccountConflict)
		}
		if adoptProvisionedBy != "" {
			if err := user.SetProvisionedBy(tx, adoptProvisionedBy); err != nil {
				return err
			}
		}
		identity.UserID = &user.ID
		return nil
	}

	// An inactive identity with no active membership stays inactive until SCIM
	// reactivation. Just-in-time sign-in must not recreate that member.
	if requireJIT && !identity.Active {
		return Fail(CodeAccessDenied)
	}
	if requireJIT && !conn.JITEnabled {
		return Fail(CodeAccessDenied)
	}
	if err := licensing.Require(ctx, p.Entitlements, conn.OrganizationID, licensing.FeatureSSO); err != nil {
		return Fail(CodeNotLicensed)
	}
	if err := billing.ValidateMemberAdmission(tx, conn.OrganizationID); err != nil {
		if seatError(err) {
			return Fail(CodeSeatLimit)
		}
		return err
	}

	if user == nil {
		created, err := models.CreateUserInTransaction(tx, conn.OrganizationID, account.ID, account.Email, firstNonEmpty(identity.Name, account.Name))
		if err != nil {
			return err
		}
		user = created
	} else {
		if user.AccountID == nil || *user.AccountID != account.ID {
			return Fail(CodeAccountConflict)
		}
		if err := user.RestoreInTransaction(tx); err != nil {
			return err
		}
	}
	if err := user.SetProvisionedBy(tx, provisionedBy); err != nil {
		return err
	}
	if err := p.Auth.AssignRoleInTransaction(tx, user.ID.String(), conn.DefaultRole, conn.OrganizationID.String(), models.DomainTypeOrganization); err != nil {
		return err
	}
	identity.UserID = &user.ID
	identity.Active = true
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// IsSeatLimit reports a billing refusal to add a member.
func IsSeatLimit(err error) bool {
	return seatError(err)
}

func seatError(err error) bool {
	if err == nil {
		return false
	}
	code, _, ok := grpcerrors.HandlerStatus(err)
	if ok && (code == codes.FailedPrecondition || code == codes.ResourceExhausted) {
		return true
	}
	statusError, ok := status.FromError(err)
	if !ok {
		return false
	}
	if statusError.Code() == codes.FailedPrecondition || statusError.Code() == codes.ResourceExhausted {
		return true
	}
	for _, detail := range statusError.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		switch info.Reason {
		case "NO_SEAT_AVAILABLE", "MEMBER_LIMIT_REACHED", "PLAN_MAXIMUM_REACHED":
			return true
		}
	}
	return false
}
