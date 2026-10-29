package runner_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

func TestPrepareCodexAuthBundleUsesAccountEmail(t *testing.T) {
	longEmail := strings.Repeat("a", 50) + "@example.com"
	for _, test := range []struct {
		name    string
		email   string
		claims  string
		idToken string
		want    string
	}{
		{name: "ID token email", claims: `{"email":"ada@example.com"}`, want: "ada@example.com"},
		{name: "long email", claims: `{"email":"` + longEmail + `"}`, want: longEmail},
		{name: "bundle email", email: "direct@example.com", claims: `{"email":"ada@example.com"}`, want: "direct@example.com"},
		{name: "missing email", claims: `{}`, want: "account-123"},
		{name: "malformed token", idToken: "not-a-jwt", want: "account-123"},
		{name: "invalid email", claims: `{"email":"not-an-email"}`, want: "account-123"},
	} {
		t.Run(test.name, func(t *testing.T) {
			idToken := test.idToken
			if idToken == "" {
				idToken = "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString([]byte(test.claims)) + ".signature"
			}
			bundle, err := json.Marshal(map[string]any{
				"email": test.email,
				"tokens": map[string]string{
					"access_token": "access-token", "refresh_token": "refresh-token",
					"id_token": idToken, "account_id": "account-123",
				},
			})
			require.NoError(t, err)
			_, label, err := runner.PrepareCodexAuthBundle(bundle)
			require.NoError(t, err)
			assert.Equal(t, test.want, label)
		})
	}
}
