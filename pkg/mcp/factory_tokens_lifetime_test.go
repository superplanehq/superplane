package mcp_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerMCPTokenLifetimePreservesRotatedGrant(t *testing.T) {
	for _, ttl := range []int{3600, 1800} {
		t.Run((time.Duration(ttl) * time.Second).String(), func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "expired-access", "refresh-1")
			calls := 0
			client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, fmt.Sprintf(`{"access_token":"fresh-access","refresh_token":"refresh-2","expires_in":%d,"token_type":"Bearer"}`, ttl))
			})
			token, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, resource, time.Hour)
			if ttl == 3600 {
				require.NoError(t, err)
				require.Equal(t, "fresh-access", token)
			} else {
				require.ErrorIs(t, err, mcp.ErrFactoryAgentResourceTokenLifetime)
			}
			require.Equal(t, "refresh-2", decryptedSecret(t, r, db, resource, models.FactoryAgentResourceSecretRefreshToken))
			// A shorter job can use the saved grant without another rotation.
			token, err = mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, resource, 15*time.Minute)
			require.NoError(t, err)
			require.Equal(t, "fresh-access", token)
			require.Equal(t, 1, calls)
		})
	}
}
