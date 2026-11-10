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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"google.golang.org/grpc/metadata"
)

func TestGatewayForwardsOnlyAuthenticatedIdentity(t *testing.T) {
	accountID := uuid.New()
	user := &models.User{ID: uuid.New(), OrganizationID: uuid.New()}
	for _, name := range []string{"organization API key", "organization member", "account"} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/identity", nil)
			for _, key := range []string{"X-User-Id", "X-Organization-Id", "X-Account-Id", "X-Token-Scopes", "X-Support-Session"} {
				request.Header.Set(key, "spoofed")
				request.Header.Set("Grpc-Metadata-"+key, "spoofed")
			}
			gateway := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(headersMatcher))
			var forwarded metadata.MD
			require.NoError(t, gateway.HandlePath(http.MethodGet, "/identity", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
				ctx, err := runtime.AnnotateIncomingContext(r.Context(), gateway, r, "/Identity")
				require.NoError(t, err)
				forwarded, _ = metadata.FromIncomingContext(ctx)
				w.WriteHeader(http.StatusNoContent)
			}))
			server := &Server{}
			handler := server.grpcGatewayHandler(gateway)
			expectedAccount := []string(nil)
			expectedUser := []string{user.ID.String()}
			expectedOrganization := []string{user.OrganizationID.String()}
			authenticatedUser := *user
			ctx := request.Context()
			if name == "organization member" {
				authenticatedUser.AccountID = &accountID
				expectedAccount = []string{accountID.String()}
			}
			ctx = context.WithValue(ctx, middleware.UserContextKey, &authenticatedUser)
			if name == "account" {
				ctx = context.WithValue(ctx, middleware.AccountContextKey, &models.Account{ID: accountID})
				handler = server.grpcGatewayAccountHandler(gateway)
				expectedAccount = []string{accountID.String()}
				expectedUser = nil
				expectedOrganization = nil
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request.WithContext(ctx))
			require.Equal(t, http.StatusNoContent, recorder.Code)
			require.Equal(t, expectedAccount, forwarded.Get("x-account-id"))
			require.Equal(t, expectedUser, forwarded.Get("x-user-id"))
			require.Equal(t, expectedOrganization, forwarded.Get("x-organization-id"))
			require.Empty(t, forwarded.Get("x-token-scopes"))
			require.Empty(t, forwarded.Get("x-support-session"))
			require.Equal(t, "spoofed", request.Header.Get("X-Account-Id"))
		})
	}
}

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
