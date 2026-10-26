package billing

import "time"

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
