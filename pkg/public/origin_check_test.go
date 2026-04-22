package public

import (
	"net/http"
	"testing"
)

func TestMakeOriginChecker(t *testing.T) {
	check := makeOriginChecker([]string{"https://app.superplane.com", "http://localhost:8000/"})

	cases := []struct {
		name   string
		origin string
		want   bool
	}{
		{"exact match", "https://app.superplane.com", true},
		{"trailing slash tolerance", "http://localhost:8000", true},
		{"case-insensitive host", "HTTPS://APP.SUPERPLANE.COM", true},
		{"different host rejected", "https://evil.example.com", false},
		{"different scheme rejected", "http://app.superplane.com", false},
		{"missing origin allowed (CLI)", "", true},
		{"malformed origin rejected", "not-a-url", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, "/ws/test", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := check(r); got != tc.want {
				t.Errorf("origin %q: got %v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}
