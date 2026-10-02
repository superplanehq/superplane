package autoapprove

import "fmt"

// Policy is the per-gate auto-approval configuration. In this stage only inert,
// low-risk changes can be auto-approved; the fields that open more categories to
// end users (the platform-team unlock and the owner opt-in) arrive in a later
// stage. Everything here is additive: a gate with no Policy behaves exactly as a
// manual approval gate.
type Policy struct {
	// InertChanges auto-approves changes classified as inert and low risk
	// (documentation, no-op). It is the only automation this stage grants.
	InertChanges bool
}

// Decision is the outcome of applying a Policy to a Classification. It is the
// single place the trust ceiling is enforced.
type Decision struct {
	AutoApprove bool
	Category    Category
	Tier        Tier
	Reason      string
}

// Decide applies the ceiling and this stage's eligibility rules. The ceiling is
// absolute and checked first: anything above Low always requires a human, and no
// Policy or installation setting can override it. installEnabled is the
// instance-level switch; whenMatched is the optional expr-lang guard, already
// evaluated by the caller (true when no guard is configured).
func Decide(cl Classification, pol Policy, installEnabled, whenMatched bool) Decision {
	base := Decision{Category: cl.Category, Tier: cl.Tier}

	// The ceiling. Mid and High are never auto-approved.
	if cl.Tier != TierLow {
		base.Reason = fmt.Sprintf("%s is %s risk; requires human approval", cl.Category, cl.Tier)
		return base
	}

	if !installEnabled {
		base.Reason = "auto-approval is disabled on this installation"
		return base
	}

	if !pol.InertChanges {
		base.Reason = "auto-approval of inert changes is not enabled on this gate"
		return base
	}

	if !cl.Inert {
		base.Reason = fmt.Sprintf("%s is low risk but not inert; requires human approval in this stage", cl.Category)
		return base
	}

	if !whenMatched {
		base.Reason = "policy guard did not match; requires human approval"
		return base
	}

	base.AutoApprove = true
	base.Reason = fmt.Sprintf("auto-approved: %s, inert low-risk change", cl.Category)
	return base
}
