package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"google.golang.org/grpc/metadata"
)

func TestGatewayMetadataPreservesAuthenticatedIdentityAndScopes(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/invitations/test/accept", nil)
	headers := map[string]string{
		"X-User-Id":         "authenticated-user",
		"X-Account-Id":      "authenticated-account",
		"X-Organization-Id": "authenticated-organization",
		"X-Token-Scopes":    `["members:create"]`,
	}
	for key, value := range headers {
		request.Header.Set(key, value)
		request.Header.Set("Grpc-Metadata-"+key, "spoofed")
	}
	gateway := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(headersMatcher))

	ctx, err := runtime.AnnotateIncomingContext(context.Background(), gateway, request, "/Organizations/AcceptInvitation")

	require.NoError(t, err)
	md, ok := metadata.FromIncomingContext(ctx)
	require.True(t, ok)
	for key, value := range headers {
		require.Equal(t, []string{value}, md.Get(strings.ToLower(key)), key)
	}
}

func TestOrganizationGatewayUsesOnlyAuthenticatedAccount(t *testing.T) {
	accountID := uuid.New()
	for _, tc := range []struct {
		name      string
		accountID *uuid.UUID
	}{
		{name: "machine"},
		{name: "human", accountID: &accountID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := &models.User{ID: uuid.New(), OrganizationID: uuid.New(), AccountID: tc.accountID}
			request := httptest.NewRequest(http.MethodPost, "/identity", nil)
			for _, key := range []string{"X-User-Id", "X-Account-Id", "X-Organization-Id", "X-Token-Scopes"} {
				request.Header.Set(key, "spoofed")
				request.Header.Set("Grpc-Metadata-"+key, "spoofed")
			}
			ctx := context.WithValue(request.Context(), middleware.UserContextKey, user)
			ctx = context.WithValue(ctx, middleware.ScopedTokenClaimsContextKey, &jwt.ScopedTokenClaims{Scopes: []string{"members:create"}})
			gateway := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(headersMatcher))
			require.NoError(t, gateway.HandlePath(http.MethodPost, "/identity", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
				ctx, err := runtime.AnnotateIncomingContext(r.Context(), gateway, r, "/Organizations/Identity")
				require.NoError(t, err)
				md, ok := metadata.FromIncomingContext(ctx)
				require.True(t, ok)
				require.Equal(t, []string{user.ID.String()}, md.Get("x-user-id"))
				require.Equal(t, []string{user.OrganizationID.String()}, md.Get("x-organization-id"))
				require.Equal(t, []string{`["members:create"]`}, md.Get("x-token-scopes"))
				if tc.accountID == nil {
					require.Empty(t, md.Get("x-account-id"))
					return
				}
				require.Equal(t, []string{accountID.String()}, md.Get("x-account-id"))
			}))
			response := httptest.NewRecorder()
			(&Server{}).grpcGatewayHandler(gateway).ServeHTTP(response, request.WithContext(ctx))
			require.Equal(t, http.StatusOK, response.Code)
		})
	}
}
