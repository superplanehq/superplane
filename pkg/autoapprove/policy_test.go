package autoapprove

import "testing"

func TestDecide_CeilingIsAbsolute(t *testing.T) {
	// Even with every permissive setting on, a change above Low is never
	// auto-approved. The ceiling ignores Policy and the install switch.
	for _, tier := range []Tier{TierMid, TierHigh} {
		cl := Classification{Category: CategoryAppCode, Tier: tier, Inert: true}
		got := Decide(cl, Policy{InertChanges: true}, true, true)
		if got.AutoApprove {
			t.Errorf("tier %q was auto-approved; the ceiling must block it", tier)
		}
	}
}

func TestDecide_InertLowHappyPath(t *testing.T) {
	cl := Classification{Category: CategoryDocs, Tier: TierLow, Inert: true}
	got := Decide(cl, Policy{InertChanges: true}, true, true)
	if !got.AutoApprove {
		t.Fatalf("inert low change was not auto-approved: %s", got.Reason)
	}
}

func TestDecide_GatedByEverySwitch(t *testing.T) {
	cl := Classification{Category: CategoryDocs, Tier: TierLow, Inert: true}

	cases := []struct {
		name        string
		pol         Policy
		installOn   bool
		whenMatched bool
		wantApprove bool
	}{
		{"install off blocks", Policy{InertChanges: true}, false, true, false},
		{"gate flag off blocks", Policy{InertChanges: false}, true, true, false},
		{"guard not matched blocks", Policy{InertChanges: true}, true, false, false},
		{"all on approves", Policy{InertChanges: true}, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Decide(cl, c.pol, c.installOn, c.whenMatched)
			if got.AutoApprove != c.wantApprove {
				t.Errorf("autoApprove = %v, want %v (%s)", got.AutoApprove, c.wantApprove, got.Reason)
			}
		})
	}
}

func TestDecide_LowButNotInertEscalates(t *testing.T) {
	cl := Classification{Category: CategoryTests, Tier: TierLow, Inert: false}
	got := Decide(cl, Policy{InertChanges: true}, true, true)
	if got.AutoApprove {
		t.Error("low but non-inert change was auto-approved; stage 1 only clears inert")
	}
}

func TestNewDecisionRecord(t *testing.T) {
	auto := NewDecisionRecord(Decision{AutoApprove: true, Category: CategoryDocs, Tier: TierLow}, "abc123", "2026-08-18T00:00:00Z")
	if auto.Decision != OutcomeAuto {
		t.Errorf("decision = %q, want %q", auto.Decision, OutcomeAuto)
	}
	if auto.Outcome != OutcomePendingDownstream {
		t.Errorf("outcome = %q, want pending", auto.Outcome)
	}
	human := NewDecisionRecord(Decision{AutoApprove: false, Category: CategoryAppCode, Tier: TierMid}, "def456", "2026-08-18T00:00:00Z")
	if human.Decision != OutcomeHumanRequired {
		t.Errorf("decision = %q, want %q", human.Decision, OutcomeHumanRequired)
	}
}
