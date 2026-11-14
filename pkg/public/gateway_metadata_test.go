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

func captureGatewayMetadata(t *testing.T, path string) (*runtime.ServeMux, *metadata.MD) {
	t.Helper()
	gateway := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(headersMatcher))
	captured := &metadata.MD{}
	require.NoError(t, gateway.HandlePath(http.MethodGet, path, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		ctx, err := runtime.AnnotateIncomingContext(r.Context(), gateway, r, "/Test/Capture")
		require.NoError(t, err)
		md, _ := metadata.FromIncomingContext(ctx)
		*captured = md
		w.WriteHeader(http.StatusOK)
	}))
	return gateway, captured
}

func TestGatewayHandlerDropsClientIdentityHeadersForAPIKeyCaller(t *testing.T) {
	const path = "/api/v1/organizations/capture"
	gateway, captured := captureGatewayMetadata(t, path)
	apiKey := &models.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Type:           models.UserTypeAPIKey,
	}

	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Account-Id", "spoofed-account")
	request.Header.Set("X-User-Id", "spoofed-user")
	request.Header.Set("X-Organization-Id", "spoofed-organization")
	request.Header.Set("X-Token-Scopes", `["*"]`)
	request.Header.Set("X-Support-Session", "true")
	request = request.WithContext(context.WithValue(request.Context(), middleware.UserContextKey, apiKey))
	recorder := httptest.NewRecorder()

	(&Server{}).grpcGatewayHandler(gateway).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, captured.Get("x-account-id"))
	require.Empty(t, captured.Get("x-token-scopes"))
	require.Empty(t, captured.Get("x-support-session"))
	require.Equal(t, []string{apiKey.ID.String()}, captured.Get("x-user-id"))
	require.Equal(t, []string{apiKey.OrganizationID.String()}, captured.Get("x-organization-id"))

	// The original request must not see gateway-only header changes.
	require.Equal(t, "spoofed-account", request.Header.Get("X-Account-Id"))
	require.Equal(t, "spoofed-organization", request.Header.Get("X-Organization-Id"))
}

func TestGatewayHandlerReplacesClientAccountIDForHumanCaller(t *testing.T) {
	const path = "/api/v1/organizations/capture"
	gateway, captured := captureGatewayMetadata(t, path)
	accountID := uuid.New()
	user := &models.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		AccountID:      &accountID,
		Type:           models.UserTypeHuman,
	}

	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Account-Id", "spoofed-account")
	request = request.WithContext(context.WithValue(request.Context(), middleware.UserContextKey, user))
	recorder := httptest.NewRecorder()

	(&Server{}).grpcGatewayHandler(gateway).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{accountID.String()}, captured.Get("x-account-id"))
}

func TestGatewayAccountHandlerDropsClientIdentityHeaders(t *testing.T) {
	const path = "/api/v1/invitations/capture"
	gateway, captured := captureGatewayMetadata(t, path)
	account := &models.Account{ID: uuid.New()}

	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Account-Id", "spoofed-account")
	request.Header.Set("X-User-Id", "spoofed-user")
	request.Header.Set("X-Organization-Id", "spoofed-organization")
	request.Header.Set("X-Token-Scopes", `["*"]`)
	request.Header.Set("X-Support-Session", "true")
	request = request.WithContext(context.WithValue(request.Context(), middleware.AccountContextKey, account))
	recorder := httptest.NewRecorder()

	(&Server{}).grpcGatewayAccountHandler(gateway).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{account.ID.String()}, captured.Get("x-account-id"))
	require.Empty(t, captured.Get("x-user-id"))
	require.Empty(t, captured.Get("x-organization-id"))
	require.Empty(t, captured.Get("x-token-scopes"))
	require.Empty(t, captured.Get("x-support-session"))
}
