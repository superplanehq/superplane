package cli

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAPIEnvironmentPairPrecedenceAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, url, token string
		valid            bool
	}{
		{"complete", "https://api.example.com/", "api-token", true},
		{"missing URL", "", "api-token", false},
		{"missing token", "https://api.example.com", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvURL, "https://legacy.example.com")
			t.Setenv(EnvToken, "legacy-token")
			t.Setenv(EnvAPIURL, tc.url)
			t.Setenv(EnvAPIToken, tc.token)
			err := ValidateEnvironmentContext()
			if !tc.valid {
				require.ErrorContains(t, err, "SUPERPLANE_API_URL and SUPERPLANE_API_TOKEN")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "https://api.example.com", GetAPIURL())
			require.Equal(t, "api-token", GetAPIToken())
		})
	}
}
