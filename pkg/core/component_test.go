package core

import "testing"

func TestOutputChannel_IsFailure(t *testing.T) {
	cases := []struct {
		label string
		want  bool
	}{
		// Known failure labels — must be classified as failure.
		{"Failure", true},
		{"Failed", true},
		{"Fail", true},
		{"Timeout", true},

		// Non-failure labels currently in use across components.
		{"Success", false},
		{"Passed", false},
		{"Approved", false},
		{"Rejected", false},
		{"Found", false},
		{"Not Found", false},
		{"Deleted", false},
		{"Default", false},
		{"True", false},
		{"False", false},
		{"Received", false},
		{"Healthy", false},
		{"Degraded", false},
		{"Clear", false},
		{"Low", false},
		{"High", false},

		// Lookup is exact-match on purpose. Case variations and
		// whitespace should NOT silently classify as failure — if a
		// caller intended a failure label but typed it differently,
		// we surface that by treating it as non-failure (and the test
		// for the failure path will fail loudly).
		{"failed", false},
		{"FAILED", false},
		{"Failed ", false},
		{" Failed", false},
		{"Request Failed", false},

		// Empty labels are non-failure.
		{"", false},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			got := OutputChannel{Label: c.label}.IsFailure()
			if got != c.want {
				t.Errorf("OutputChannel{Label: %q}.IsFailure() = %v, want %v", c.label, got, c.want)
			}
		})
	}
}
