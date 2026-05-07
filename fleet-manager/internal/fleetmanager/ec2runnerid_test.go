package fleetmanager

import "testing"

func TestIsEC2InstanceID(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"i-0abcdef1234567890", true},
		{"i-01", false},
		{"", false},
		{"hostname", false},
		{"I-0ABCDEF1234567890", false},
	}
	for _, tc := range cases {
		if got := isEC2InstanceID(tc.s); got != tc.want {
			t.Errorf("isEC2InstanceID(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}
