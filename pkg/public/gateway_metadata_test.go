package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
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
