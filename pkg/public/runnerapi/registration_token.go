package runnerapi

import "github.com/superplanehq/superplane/pkg/runners/registration"

const RegistrationAudience = registration.RegistrationAudience

type RegistrationClaims = registration.RegistrationClaims

var (
	ErrInvalidRegistrationToken = registration.ErrInvalidRegistrationToken
	MintRegistrationToken       = registration.MintRegistrationToken
	ValidateRegistrationToken   = registration.ValidateRegistrationToken
)
