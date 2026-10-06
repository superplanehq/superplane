package licensing

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type Edition string

const (
	EditionCommunity  Edition = "community"
	EditionEnterprise Edition = "enterprise"
)

type Validity string

const (
	ValidityActive      Validity = "active"
	ValidityNotYetValid Validity = "not_yet_valid"
	ValidityExpired     Validity = "expired"
)

// License is the trusted result of a successful signature and claims
// verification. It never contains the raw artifact.
type License struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	KeyID      string
	Issuer     string
	Edition    Edition

	// Features contains only keys that this SuperPlane version recognizes.
	Features  []Feature
	IssuedAt  time.Time
	ValidFrom time.Time
	ExpiresAt time.Time
}

func (l *License) HasFeature(feature Feature) bool {
	if l == nil {
		return false
	}

	return slices.Contains(l.Features, feature)
}

func (l *License) ValidityAt(now time.Time) Validity {
	if now.Add(ClockSkew).Before(l.ValidFrom) {
		return ValidityNotYetValid
	}

	if !now.Add(-ClockSkew).Before(l.ExpiresAt) {
		return ValidityExpired
	}

	return ValidityActive
}
