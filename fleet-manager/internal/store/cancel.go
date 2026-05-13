package store

// CancelOutcome is the result of RequestCancelTask.
type CancelOutcome string

const (
	CancelOutcomeNotFound        CancelOutcome = "not_found"
	CancelOutcomeAlreadyTerminal CancelOutcome = "already_terminal"
	CancelOutcomeCanceledQueued  CancelOutcome = "canceled"
	CancelOutcomeCancelRequested CancelOutcome = "cancel_requested"
)
