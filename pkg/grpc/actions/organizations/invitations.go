package organizations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type InvitationService struct {
	AuthService  authorization.Authorization
	EmailService services.EmailService
	Entitlements licensing.Entitlements
	BaseURL      string
}

func (s *InvitationService) Create(ctx context.Context, orgID, inviterID string, req *pb.CreateInvitationRequest) (*pb.CreateInvitationResponse, error) {
	organization, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}
	inviter, err := uuid.Parse(inviterID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid inviter ID.")
	}
	email := utils.NormalizeEmail(req.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 255 {
		return nil, grpcerrors.InvalidArgument(nil, "Enter a valid email address.")
	}
	role := req.Role
	if role == "" {
		role = models.RoleOrgOperator
	}
	if err := s.validateRole(ctx, orgID, role); err != nil {
		return nil, err
	}
	if err := s.authorizeInvitationRole(ctx, orgID, inviterID, role); err != nil {
		return nil, err
	}
	invitation := &models.OrganizationInvitation{OrganizationID: organization, Email: email, Role: role, InvitedBy: inviter}
	var token string
	var org *models.Organization
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		org, err = models.LockOrganization(tx, organization)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := models.ExpireOrganizationInvitations(tx, organization, now); err != nil {
			return err
		}
		if err := validateInvitationRecipient(tx, invitation); err != nil {
			return err
		}
		if err := requireInvitationCapacity(tx, organization, false); err != nil {
			return err
		}
		token, err = prepareInvitation(tx, invitation, now)
		if err != nil {
			return err
		}
		if err := invitation.Save(tx); err != nil {
			return err
		}
		// Load the inviter name in the same transaction as the invitation.
		loaded, err := models.FindOrganizationInvitation(tx, organization, invitation.ID)
		if err != nil {
			return err
		}
		invitation.InviterName = loaded.InviterName
		return nil
	})
	if err != nil {
		return nil, invitationActionError(err)
	}
	acceptanceURL := s.acceptanceURL(token)
	s.sendEmail(invitation, org, acceptanceURL)
	return &pb.CreateInvitationResponse{Invitation: invitationToProto(invitation), AcceptanceUrl: acceptanceURL}, nil
}

func (s *InvitationService) List(ctx context.Context, orgID string) (*pb.ListInvitationsResponse, error) {
	organization, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}
	var response pb.ListInvitationsResponse
	// Keep the list and seat summary consistent with invitation mutations.
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, organization); err != nil {
			return err
		}
		invitations, err := models.ListOrganizationInvitations(tx, organization)
		if err != nil {
			return err
		}
		seats, err := entitlements.ResolveSeats(tx, organization)
		if err != nil {
			return err
		}
		settings, err := models.GetInstallationMetadata(tx)
		if err != nil {
			return err
		}
		response.InvitationDurationDays = int32(settings.InvitationDurationDays)
		response.Invitations = make([]*pb.OrganizationInvitation, 0, len(invitations))
		for i := range invitations {
			response.Invitations = append(response.Invitations, invitationToProto(&invitations[i]))
		}
		response.Seats = &pb.InvitationSeatSummary{PerSeat: seats.PerSeat, Purchased: seatCountToProto(seats.Purchased), Occupied: int64(seats.Occupied), PendingInvitations: int64(seats.PendingInvitations), Invitable: seatCountToProto(seats.Invitable), Purchasable: seatCountToProto(seats.Purchasable), MemberLimit: seatCountToProto(seats.MemberLimit)}
		return nil
	})
	if err != nil {
		return nil, invitationActionError(err)
	}
	return &response, nil
}

func (s *InvitationService) Resend(ctx context.Context, orgID, actorID, invitationID string) (*pb.ResendInvitationResponse, error) {
	organization, id, err := parseInvitationIDs(orgID, invitationID)
	if err != nil {
		return nil, err
	}
	var invitation *models.OrganizationInvitation
	var org *models.Organization
	var token string
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		org, err = models.LockOrganization(tx, organization)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := models.ExpireOrganizationInvitations(tx, organization, now); err != nil {
			return err
		}
		invitation, err = models.FindOrganizationInvitation(tx, organization, id)
		if err != nil {
			return err
		}
		if invitation.State != models.InvitationStatePending && invitation.State != models.InvitationStateExpired {
			return grpcerrors.FailedPrecondition(nil, "Only pending or expired invitations can be resent.")
		}
		if err := s.validateRole(ctx, orgID, invitation.Role); err != nil {
			return err
		}
		if err := s.authorizeInvitationRole(ctx, orgID, actorID, invitation.Role); err != nil {
			return err
		}
		if err := requireInvitationNonmember(tx, invitation); err != nil {
			return err
		}
		if invitation.State == models.InvitationStateExpired {
			if err := requireNoPendingInvitation(tx, invitation); err != nil {
				return err
			}
		}
		if err := requireInvitationCapacity(tx, organization, invitation.State == models.InvitationStatePending); err != nil {
			return err
		}
		token, err = prepareInvitation(tx, invitation, now)
		if err != nil {
			return err
		}
		return invitation.Save(tx)
	})
	if err != nil {
		return nil, invitationActionError(err)
	}
	acceptanceURL := s.acceptanceURL(token)
	s.sendEmail(invitation, org, acceptanceURL)
	return &pb.ResendInvitationResponse{Invitation: invitationToProto(invitation), AcceptanceUrl: acceptanceURL}, nil
}

func (s *InvitationService) Cancel(ctx context.Context, orgID, invitationID string) (*pb.CancelInvitationResponse, error) {
	organization, id, err := parseInvitationIDs(orgID, invitationID)
	if err != nil {
		return nil, err
	}
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, organization); err != nil {
			return err
		}
		invitation, err := models.FindOrganizationInvitation(tx, organization, id)
		if err != nil {
			return err
		}
		if invitation.State == models.InvitationStateAccepted {
			return grpcerrors.FailedPrecondition(nil, "Accepted invitations cannot be canceled.")
		}
		if invitation.State == models.InvitationStateCanceled {
			return nil
		}
		invitation.State = models.InvitationStateCanceled
		invitation.TokenHash = nil
		return invitation.Save(tx)
	})
	if err != nil {
		return nil, invitationActionError(err)
	}
	return &pb.CancelInvitationResponse{}, nil
}

func (s *InvitationService) authorizeInvitationRole(ctx context.Context, orgID, actorID, role string) error {
	if role == models.RoleOrgOperator {
		return nil
	}
	allowed, err := s.AuthService.CheckOrganizationPermission(ctx, actorID, orgID, "members", "update")
	if err != nil {
		return grpcerrors.Internal(err, "Failed to check invitation role permissions.")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	tokenScopes := ""
	if values := md.Get("x-token-scopes"); len(values) > 0 {
		tokenScopes = values[0]
	}
	if !allowed || !authorization.HasRequiredScopedTokenPermissionForScopes(tokenScopes, nil, authorization.AuthorizationRule{Resource: "members", Action: "update"}) {
		return grpcerrors.PermissionDenied(nil, "You do not have permission to invite members with this role.")
	}
	return nil
}

func (s *InvitationService) validateRole(ctx context.Context, orgID, role string) error {
	if _, err := s.AuthService.GetRoleDefinition(ctx, role, models.DomainTypeOrganization, orgID); err != nil {
		return grpcerrors.InvalidArgument(err, "Select an existing organization role.")
	}
	if !s.AuthService.IsDefaultRole(role, models.DomainTypeOrganization) && !licensing.Allows(s.Entitlements, licensing.FeatureCustomRoles) {
		return grpcerrors.PermissionDenied(licensing.ErrNotLicensed, licensing.ErrNotLicensed.Error())
	}
	return nil
}

func validateInvitationRecipient(tx *gorm.DB, invitation *models.OrganizationInvitation) error {
	if err := requireInvitationNonmember(tx, invitation); err != nil {
		return err
	}
	return requireNoPendingInvitation(tx, invitation)
}

func requireInvitationNonmember(tx *gorm.DB, invitation *models.OrganizationInvitation) error {
	user, err := models.FindMaybeDeletedUserByEmailInTransaction(tx, invitation.OrganizationID.String(), invitation.Email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !user.DeletedAt.Valid {
		return grpcerrors.AlreadyExists(nil, "This email address belongs to an active member.")
	}
	return nil
}

func requireNoPendingInvitation(tx *gorm.DB, invitation *models.OrganizationInvitation) error {
	pending, err := models.HasPendingOrganizationInvitation(tx, invitation.OrganizationID, invitation.Email)
	if err != nil {
		return err
	}
	if pending {
		return grpcerrors.AlreadyExists(nil, "A pending invitation already exists for this email address.")
	}
	return nil
}

func requireInvitationCapacity(tx *gorm.DB, orgID uuid.UUID, alreadyPending bool) error {
	seats, err := entitlements.ResolveSeats(tx, orgID)
	if err != nil {
		return err
	}
	if seats.PaymentBlocked {
		return grpcerrors.FailedPreconditionWithReason("The last renewal payment failed. Update the payment method before you invite members.", "PAYMENT_PAST_DUE")
	}
	if seats.PerSeat {
		if alreadyPending || *seats.Invitable > 0 {
			return nil
		}
		if seats.Purchasable != nil && *seats.Purchasable == 0 {
			return grpcerrors.FailedPreconditionWithReason("This organization has reached its plan seat maximum.", "PLAN_MAXIMUM_REACHED")
		}
		return grpcerrors.FailedPreconditionWithReason("Purchase a seat before sending another invitation.", "SEAT_PURCHASE_REQUIRED")
	}
	if seats.MemberLimit != nil && seats.Occupied >= *seats.MemberLimit {
		return grpcerrors.FailedPreconditionWithReason("This organization has reached its member limit.", "MEMBER_LIMIT_REACHED")
	}
	return nil
}

func prepareInvitation(tx *gorm.DB, invitation *models.OrganizationInvitation, now time.Time) (string, error) {
	// Match PostgreSQL precision so the response and refreshed list identify the same link.
	now = now.UTC().Truncate(time.Microsecond)
	settings, err := models.GetInstallationMetadata(tx)
	if err != nil {
		return "", err
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	hash := crypto.HashToken(token)
	invitation.TokenHash = &hash
	invitation.State = models.InvitationStatePending
	invitation.SentAt = now
	invitation.ExpiresAt = now.Add(time.Duration(settings.InvitationDurationDays) * 24 * time.Hour)
	return token, nil
}

func (s *InvitationService) acceptanceURL(token string) string {
	return strings.TrimRight(s.BaseURL, "/") + "/invitations/" + token
}

func (s *InvitationService) sendEmail(invitation *models.OrganizationInvitation, org *models.Organization, acceptanceURL string) {
	if s.EmailService == nil {
		return
	}
	err := s.EmailService.SendOrganizationInvitationEmail(invitation.Email, services.OrganizationInvitationTemplateData{OrganizationName: org.Name, InviterName: invitation.InviterName, Role: invitation.Role, ExpiresAt: invitation.ExpiresAt.UTC().Format(time.RFC1123), AcceptanceURL: acceptanceURL})
	if err != nil {
		log.WithError(err).WithField("invitation_id", invitation.ID).Error("Failed to send organization invitation email")
	}
}

func invitationToProto(invitation *models.OrganizationInvitation) *pb.OrganizationInvitation {
	return &pb.OrganizationInvitation{Id: invitation.ID.String(), Email: invitation.Email, Role: invitation.Role, InvitedBy: invitation.InvitedBy.String(), InviterName: invitation.InviterName, State: invitation.EffectiveState(time.Now()), SentAt: timestamppb.New(invitation.SentAt), ExpiresAt: timestamppb.New(invitation.ExpiresAt)}
}

func seatCountToProto(count *int) *int64 {
	if count == nil {
		return nil
	}
	value := int64(*count)
	return &value
}

func parseInvitationIDs(orgID, invitationID string) (uuid.UUID, uuid.UUID, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}
	id, err := uuid.Parse(invitationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, grpcerrors.InvalidArgument(err, "Invalid invitation ID.")
	}
	return org, id, nil
}

func invitationActionError(err error) error {
	if grpcerrors.Code(err) != codes.Unknown {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return grpcerrors.NotFound(err, "Invitation or organization not found.")
	}
	return grpcerrors.Internal(err, "Failed to manage organization invitations.")
}
