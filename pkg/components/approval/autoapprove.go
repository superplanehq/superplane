package approval

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/autoapprove"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// ItemTypeAuto marks an approval record that was granted by policy rather than by
// a person. It shares the same Record shape as a human decision so the audit
// trail is uniform: every clearance, human or automatic, is one record with a
// reason.
const ItemTypeAuto = "auto"

// autoApproveInstallEnabledEnv is the instance-level switch. Auto-approval of
// inert changes is off unless an operator turns it on, so the feature never
// changes behavior on an existing installation by default.
const autoApproveInstallEnabledEnv = "AUTO_APPROVE_INERT_ENABLED"

// AutoApprovePolicy is the optional auto-approval configuration on an approval
// gate. A gate with no policy behaves exactly as a manual approval gate. In this
// stage the only automation is clearing inert, low-risk changes; mid and high
// risk always require a human and no setting can override that.
type AutoApprovePolicy struct {
	// InertChanges auto-approves changes classified as inert and low risk.
	InertChanges bool `json:"inertChanges" mapstructure:"inertChanges"`

	// When is an optional expr-lang guard evaluated against the change. When set,
	// it must be true for auto-approval to apply. Empty means "no extra guard".
	When string `json:"when,omitempty" mapstructure:"when"`

	// Reason is optional human-facing text recorded with an automatic approval.
	Reason string `json:"reason,omitempty" mapstructure:"reason"`
}

// tryAutoApprove classifies the incoming change, records the decision in the
// ledger, and, when the change is eligible, emits an approval and reports that it
// handled the execution. It returns (false, nil) when there is no policy or the
// change must go to a human, so the caller falls through to the manual flow.
func (a *Approval) tryAutoApprove(ctx core.ExecutionContext, config *Config) (bool, error) {
	if config.AutoApprove == nil {
		return false, nil
	}

	change := autoapprove.ChangeFromPayload(ctx.Data)
	classification := autoapprove.Classify(change)
	policy := autoapprove.Policy{InertChanges: config.AutoApprove.InertChanges}
	installEnabled := autoApproveInstallEnabled()

	// Evaluate the optional guard only when the change would otherwise be
	// auto-approved, and never let a guard failure block the gate: on any error
	// or a non-boolean result, escalate to a human rather than failing the node.
	whenMatched := true
	if strings.TrimSpace(config.AutoApprove.When) != "" &&
		autoapprove.Decide(classification, policy, installEnabled, true).AutoApprove {
		whenMatched = a.evalGuard(ctx, config.AutoApprove.When)
	}

	decision := autoapprove.Decide(classification, policy, installEnabled, whenMatched)

	// Record every decision, automatic or escalated, so the ledger is complete
	// and later stages can bind outcomes to it. A ledger write failure must not
	// block the gate, so it is logged rather than returned.
	record := autoapprove.NewDecisionRecord(decision, autoApproveCorrelation(ctx), time.Now().UTC().Format(time.RFC3339))
	if ctx.CanvasMemory != nil {
		if err := ctx.CanvasMemory.Add(autoapprove.LedgerNamespace, record); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("approval: failed to write decision ledger: %v", err)
		}
	}

	if !decision.AutoApprove {
		return false, nil
	}

	comment := decision.Reason
	if config.AutoApprove.Reason != "" {
		comment = config.AutoApprove.Reason + " (" + decision.Reason + ")"
	}

	metadata := &Metadata{Records: []Record{{
		Index: 0,
		Type:  ItemTypeAuto,
		State: StateApproved,
		Approval: &ApprovalInfo{
			ApprovedAt: time.Now().UTC().Format(time.RFC3339),
			Comment:    comment,
		},
	}}}
	metadata.UpdateResult()

	if err := ctx.Metadata.Set(metadata); err != nil {
		return false, fmt.Errorf("error setting auto-approval metadata: %w", err)
	}

	return true, ctx.ExecutionState.Emit(ChannelApproved, "approval.auto_approved", []any{metadata})
}

// evalGuard runs the optional guard expression. Any problem (missing evaluator,
// an evaluation error, or a non-boolean result) returns false so the change goes
// to a human, keeping the failure mode on the safe side.
func (a *Approval) evalGuard(ctx core.ExecutionContext, expr string) bool {
	if ctx.Expressions == nil {
		return false
	}
	out, err := ctx.Expressions.Run(expr)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("approval: auto-approve guard failed to evaluate, escalating: %v", err)
		}
		return false
	}
	matched, ok := out.(bool)
	if !ok {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("approval: auto-approve guard did not return a boolean (%T), escalating", out)
		}
		return false
	}
	return matched
}

func autoApproveInstallEnabled() bool {
	return os.Getenv(autoApproveInstallEnabledEnv) == "yes"
}

// autoApproveCorrelation returns a stable key that links this decision to the
// change it cleared, so a downstream revert, rollback, or incident can be
// attached to it in a later stage. It prefers an id from the payload and falls
// back to the execution id.
func autoApproveCorrelation(ctx core.ExecutionContext) string {
	if m, ok := ctx.Data.(map[string]any); ok {
		for _, key := range []string{"sha", "commit_sha", "pull_request_number", "pr_number", "id"} {
			if v, ok := m[key]; ok && v != nil {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ctx.ID.String()
}

// autoApproveField is the optional configuration block rendered on the approval
// gate. It is appended to the gate's fields; leaving it unset keeps the gate
// manual.
func autoApproveField() configuration.Field {
	return configuration.Field{
		Name:        "autoApprove",
		Label:       "Auto-approval policy",
		Description: "Optional. Automatically clears inert, low-risk changes. Mid and high risk always require a human.",
		Type:        configuration.FieldTypeObject,
		Required:    false,
		Togglable:   true,
		TypeOptions: &configuration.TypeOptions{
			Object: &configuration.ObjectTypeOptions{
				Schema: []configuration.Field{
					{
						Name:        "inertChanges",
						Label:       "Auto-approve inert changes",
						Description: "Clear documentation and no-op changes without a human.",
						Type:        configuration.FieldTypeBool,
						Default:     false,
					},
					{
						Name:        "when",
						Label:       "Guard expression",
						Description: "Optional. Must evaluate to true for auto-approval to apply.",
						Type:        configuration.FieldTypeExpression,
						Required:    false,
					},
					{
						Name:        "reason",
						Label:       "Reason",
						Description: "Optional text recorded with an automatic approval.",
						Type:        configuration.FieldTypeString,
						Required:    false,
					},
				},
			},
		},
	}
}
