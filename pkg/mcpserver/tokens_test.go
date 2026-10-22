package mcpserver

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	claims := AccessClaims{
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
		FactoryIDs: []uuid.UUID{uuid.New(), uuid.New()},
		ClientID:   "superplane-local",
		Resource:   "http://localhost:8000/mcp",
		Scopes:     GrantedScopes,
	}
	token, err := MintAccessToken(signer, claims, time.Hour)
	require.NoError(t, err)

	parsed, err := ParseAccessToken(signer, token, claims.Resource)
	require.NoError(t, err)
	require.Equal(t, claims.UserID, parsed.UserID)
	require.Equal(t, claims.OrgID, parsed.OrgID)
	require.Equal(t, claims.FactoryIDs, parsed.FactoryIDs)
	require.Equal(t, claims.ClientID, parsed.ClientID)
	require.Equal(t, claims.Resource, parsed.Resource)
	require.Equal(t, GrantedScopes, parsed.Scopes)
}

func TestAccessTokenRejectsWrongAudience(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	token, err := MintAccessToken(signer, AccessClaims{
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
		FactoryIDs: []uuid.UUID{uuid.New()},
		ClientID:   "superplane-local",
		Resource:   "http://localhost:8000/mcp",
		Scopes:     GrantedScopes,
	}, time.Hour)
	require.NoError(t, err)

	_, err = ParseAccessToken(signer, token, "https://other.example/mcp")
	require.Error(t, err)
}

func TestAccessTokenRequiresClientID(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	_, err := MintAccessToken(signer, AccessClaims{
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
		FactoryIDs: []uuid.UUID{uuid.New()},
		Resource:   "http://localhost:8000/mcp",
		Scopes:     GrantedScopes,
	}, time.Hour)
	require.Error(t, err)
}

func TestAccessTokenRejectsPlanningSessionPurpose(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":    "planning_session",
		"sub":        uuid.New().String(),
		"org_id":     uuid.New().String(),
		"factory_id": uuid.New().String(),
		"aud":        "http://localhost:8000/mcp",
		"scope":      "work_orders:read",
	})
	require.NoError(t, err)

	_, err = ParseAccessToken(signer, token, "http://localhost:8000/mcp")
	require.Error(t, err)
}

func TestAccessTokenDoesNotValidateAsScopedToken(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	token, err := MintAccessToken(signer, AccessClaims{
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
		FactoryIDs: []uuid.UUID{uuid.New()},
		ClientID:   "superplane-local",
		Resource:   "http://localhost:8000/mcp",
		Scopes:     GrantedScopes,
	}, time.Hour)
	require.NoError(t, err)

	_, err = signer.ValidateScopedToken(token)
	require.Error(t, err)
}

func TestAccessTokenReadsLegacySingleWorkspaceClaim(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	factoryID := uuid.New()
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":    AccessTokenPurpose,
		"sub":        uuid.NewString(),
		"org_id":     uuid.NewString(),
		"factory_id": factoryID.String(),
		"client_id":  "superplane-local",
		"aud":        "http://localhost:8000/mcp",
		"scope":      "work_orders:read",
	})
	require.NoError(t, err)

	parsed, err := ParseAccessToken(signer, token, "http://localhost:8000/mcp")
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{factoryID}, parsed.FactoryIDs)
}

func TestAccessTokenRejectsInvalidWorkspaceList(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	tests := map[string]any{
		"empty list":     []string{},
		"malformed id":   []string{"not-a-uuid"},
		"nil id":         []string{uuid.Nil.String()},
		"not a list":     uuid.NewString(),
		"one bad member": []string{uuid.NewString(), ""},
	}
	for name, factoryIDs := range tests {
		t.Run(name, func(t *testing.T) {
			token, err := signer.GenerateWithJSONClaims(time.Hour, map[string]any{
				"purpose":     AccessTokenPurpose,
				"sub":         uuid.NewString(),
				"org_id":      uuid.NewString(),
				"factory_ids": factoryIDs,
				"client_id":   "superplane-local",
				"aud":         "http://localhost:8000/mcp",
				"scope":       "work_orders:read",
			})
			require.NoError(t, err)

			_, err = ParseAccessToken(signer, token, "http://localhost:8000/mcp")
			require.Error(t, err)
		})
	}
}

func TestAccessTokenRejectsMissingWorkspace(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":   AccessTokenPurpose,
		"sub":       uuid.NewString(),
		"org_id":    uuid.NewString(),
		"client_id": "superplane-local",
		"aud":       "http://localhost:8000/mcp",
		"scope":     "work_orders:read",
	})
	require.NoError(t, err)

	_, err = ParseAccessToken(signer, token, "http://localhost:8000/mcp")
	require.Error(t, err)
}

func TestAccessTokenRequiresWorkspace(t *testing.T) {
	signer := jwt.NewSigner("test-secret")
	for name, factoryIDs := range map[string][]uuid.UUID{"none": nil, "nil id": {uuid.Nil}} {
		t.Run(name, func(t *testing.T) {
			_, err := MintAccessToken(signer, AccessClaims{
				UserID:     uuid.New(),
				OrgID:      uuid.New(),
				FactoryIDs: factoryIDs,
				ClientID:   "superplane-local",
				Resource:   "http://localhost:8000/mcp",
				Scopes:     GrantedScopes,
			}, time.Hour)
			require.Error(t, err)
		})
	}
}
