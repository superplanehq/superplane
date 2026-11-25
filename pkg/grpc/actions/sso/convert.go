package sso

import (
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/sso"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func protocolToProto(value string) pb.Protocol {
	if value == models.SSOProtocolOIDC {
		return pb.Protocol_PROTOCOL_OIDC
	}
	return pb.Protocol_PROTOCOL_UNSPECIFIED
}

func presetToProto(value string) pb.Preset {
	switch value {
	case models.SSOPresetOkta:
		return pb.Preset_PRESET_OKTA
	case models.SSOPresetEntra:
		return pb.Preset_PRESET_ENTRA
	case models.SSOPresetGeneric:
		return pb.Preset_PRESET_GENERIC
	default:
		return pb.Preset_PRESET_UNSPECIFIED
	}
}

func presetFromProto(value pb.Preset) string {
	switch value {
	case pb.Preset_PRESET_OKTA:
		return models.SSOPresetOkta
	case pb.Preset_PRESET_ENTRA:
		return models.SSOPresetEntra
	default:
		return models.SSOPresetGeneric
	}
}

func statusToProto(value string) pb.Status {
	switch value {
	case models.SSOStatusDraft:
		return pb.Status_STATUS_DRAFT
	case models.SSOStatusActive:
		return pb.Status_STATUS_ACTIVE
	case models.SSOStatusDisabled:
		return pb.Status_STATUS_DISABLED
	default:
		return pb.Status_STATUS_UNSPECIFIED
	}
}

func enforcementToProto(value string) pb.Enforcement {
	switch value {
	case models.SSOEnforcementRequired:
		return pb.Enforcement_ENFORCEMENT_REQUIRED
	case models.SSOEnforcementOptional:
		return pb.Enforcement_ENFORCEMENT_OPTIONAL
	default:
		return pb.Enforcement_ENFORCEMENT_UNSPECIFIED
	}
}

func enforcementFromProto(value pb.Enforcement) string {
	if value == pb.Enforcement_ENFORCEMENT_REQUIRED {
		return models.SSOEnforcementRequired
	}
	return models.SSOEnforcementOptional
}

func connectionToProto(view *enterprise.SSOConnection) *pb.Connection {
	if view == nil || !view.Exists {
		return nil
	}
	connection := &pb.Connection{
		Id:                       view.ID.String(),
		OrganizationId:           view.OrganizationID.String(),
		Protocol:                 protocolToProto(view.Protocol),
		Preset:                   presetToProto(view.Preset),
		DisplayName:              view.DisplayName,
		Status:                   statusToProto(view.Status),
		Enforcement:              enforcementToProto(view.Enforcement),
		AllowOwnerBypass:         view.AllowOwnerBypass,
		JitEnabled:               view.JITEnabled,
		DefaultRole:              view.DefaultRole,
		MaxSessionAgeSeconds:     view.MaxSessionAgeSeconds,
		IdpLogoutEnabled:         view.IDPLogoutEnabled,
		Issuer:                   view.Issuer,
		ClientId:                 view.ClientID,
		Scopes:                   view.Scopes,
		SubjectClaim:             view.SubjectClaim,
		EmailClaim:               view.EmailClaim,
		NameClaim:                view.NameClaim,
		ClientSecretSet:          view.ClientSecretSet,
		RedirectUri:              view.RedirectURI,
		InitiateLoginUri:         view.InitiateLoginURI,
		PostLogoutRedirectUri:    view.PostLogoutRedirectURI,
		BackChannelLogoutUri:     view.BackChannelLogoutURI,
		ScimBaseUrl:              view.SCIMBaseURL,
		GlobalTokenRevocationUri: view.GlobalTokenRevocationURI,
	}
	if view.LastTestedAt != nil {
		connection.LastTestedAt = timestamppb.New(*view.LastTestedAt)
	}
	if view.LastTest != nil {
		connection.LastTest = &pb.TestResult{
			Ok:      view.LastTest.OK,
			Error:   view.LastTest.Error,
			Subject: view.LastTest.Subject,
			Email:   view.LastTest.Email,
			Name:    view.LastTest.Name,
		}
	}
	return connection
}

func domainToProto(view enterprise.SSODomain) *pb.Domain {
	domain := &pb.Domain{
		Id:                 view.ID.String(),
		Domain:             view.Domain,
		VerificationToken:  view.VerificationToken,
		TxtName:            view.TXTName,
		TxtValue:           view.TXTValue,
		VerificationMethod: view.VerificationMethod,
	}
	if view.VerifiedAt != nil {
		domain.VerifiedAt = timestamppb.New(*view.VerifiedAt)
	}
	return domain
}
