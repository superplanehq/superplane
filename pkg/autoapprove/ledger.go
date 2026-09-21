package autoapprove

// LedgerNamespace is the canvas-memory namespace the decision ledger is written
// to. Every decision, automatic or escalated to a human, is appended here so the
// question "what cleared without a person, and why" can always be answered, and
// so later stages can bind outcomes back to decisions.
const LedgerNamespace = "approval_ledger"

// Decision outcomes recorded in the ledger.
const (
	OutcomeAuto              = "auto"
	OutcomeHumanRequired     = "human_required"
	OutcomePendingDownstream = "pending" // set until an outcome is bound in a later stage
)

// DecisionRecord is one row of the ledger. It is a plain value so it can be
// stored through the canvas-memory API without any new schema, and read back for
// the per-category reports. Correlation links a decision to the change it cleared
// (a commit SHA, a pull-request number, a deploy id) so a downstream revert,
// rollback, or incident can be attached to it later.
type DecisionRecord struct {
	Category    string `json:"category"`
	Tier        string `json:"tier"`
	Decision    string `json:"decision"`
	Correlation string `json:"correlation"`
	Reason      string `json:"reason"`
	DecidedAt   string `json:"decidedAt"`
	Outcome     string `json:"outcome"`
}

// NewDecisionRecord builds a ledger row from a Decision. decidedAt and correlation
// are supplied by the caller, which has the execution context and the clock.
func NewDecisionRecord(d Decision, correlation, decidedAt string) DecisionRecord {
	decision := OutcomeHumanRequired
	if d.AutoApprove {
		decision = OutcomeAuto
	}
	return DecisionRecord{
		Category:    string(d.Category),
		Tier:        string(d.Tier),
		Decision:    decision,
		Correlation: correlation,
		Reason:      d.Reason,
		DecidedAt:   decidedAt,
		Outcome:     OutcomePendingDownstream,
	}
}
