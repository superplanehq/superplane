package billing

import (
	"time"

	"github.com/google/uuid"
)

// Rate contains cents per million tokens from a provider catalog.
type Rate struct {
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
	Reasoning  int64
}

type Usage struct {
	Provider         string
	Model            string
	FundingSource    string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	CostMicros       *int64
}

type ComputeUsage struct {
	MachineType     string
	FleetID         string
	DurationSeconds int64
}

type UsagePrice struct {
	ProviderCostMicros int64
	CostMicros         int64
	PriceBookVersion   string
}

type SeatUsage struct {
	PendingInvitations int  `json:"pending_invitations"`
	Invitable          *int `json:"invitable"`
	PerSeat            bool `json:"per_seat"`
	Purchased          *int `json:"purchased"`
	Occupied           int  `json:"occupied"`
	Available          *int `json:"available"`
	PlanMaximum        *int `json:"plan_maximum"`
	Purchasable        *int `json:"purchasable"`
	RenewalQuantity    *int `json:"renewal_quantity"`
	MemberLimit        *int `json:"member_limit"`
	PaymentBlocked     bool `json:"payment_blocked"`
	// AdmissionLimit is the seat count that new members and invitations use.
	// It can be lower than Purchased while a seat removal is pending. Paid
	// quantity and the purchase maximum stay on Purchased and Purchasable.
	AdmissionLimit *int `json:"admission_limit,omitempty"`
	// AdmissionBlock is set when new seat use is stopped until the current
	// billing period is verified. Empty means AdmissionLimit is in force.
	AdmissionBlock string `json:"admission_block,omitempty"`
}

// SeatQuantityView is the removal state that Billing controls read. It carries
// no billing rule of its own.
type SeatQuantityView struct {
	OrganizationID     uuid.UUID
	SeatChangeVersion  int64
	OperationID        string
	Revision           int
	Origin             string
	Action             string
	Target             *int
	EffectiveAt        *time.Time
	Status             string
	AllowedActions     []string
	Reasons            []string
	SourcePaidQuantity *int
	SubscriptionID     string
	PeriodStart        *time.Time
	PeriodEnd          *time.Time
	MemberFloor        int
	Cutoff             *time.Time
	Binding            SeatQuantityBinding
}

// SeatQuantityBinding is the preview context a later confirmation must send
// back. An empty PendingOperationID is the versioned absence of a removal.
type SeatQuantityBinding struct {
	OrganizationID     uuid.UUID
	SubscriptionID     string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	SourcePaidQuantity int
	Target             int
	SeatChangeVersion  int64
	PendingOperationID string
	PendingRevision    int
	ImmediateAdmission int
	RenewalQuantity    int
	MemberFloor        int
	EffectiveAt        time.Time
	Cutoff             time.Time
	CurrentChargeCents int64
	RefundCents        int64
	CreditCents        int64
	NoCurrentCharge    bool
}

// SeatQuantityPreview is the read-only description of one proposed removal.
type SeatQuantityPreview struct {
	SeatQuantityBinding
	Allowed bool
	Reason  string
	Message string
}

type CreditSummary struct {
	GrantMicros              int64
	SuperPlaneGrantMicros    int64
	PurchasedCreditMicros    int64
	BilledMicros             int64
	RemainingMicros          int64
	IncludedRemainingMicros  int64
	PurchasedRemainingMicros int64
	WelcomeRemainingMicros   int64
	AdminRemainingMicros     int64
	MarkupBPS                int
	Warning                  bool
	WelcomeCreditExpiresAt   *time.Time
}

type FactoryBudget struct {
	BudgetCents     *int64
	BilledMicros    int64
	RemainingMicros int64
	Warning         bool
	Capped          bool
}

// MicrosToCents formats a nonnegative ledger amount in whole cents.
func MicrosToCents(micros int64) int64 {
	if micros < 0 {
		return 0
	}
	return micros / 10_000
}
