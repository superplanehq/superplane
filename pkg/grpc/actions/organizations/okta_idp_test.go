package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetOktaIdpSettings_Unconfigured(t *testing.T) {
	r := support.Setup(t)

	resp, err := GetOktaIdpSettings(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, resp.Settings)
	assert.Equal(t, r.Organization.ID.String(), resp.Settings.OrganizationId)
	assert.False(t, resp.Settings.Configured)
}

func Test__UpdateOktaIdpSettings_CreateAndPatch(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	issuer := "https://dev-example.okta.com/oauth2/default"
	clientID := "client-id-1"
	secret := "super-secret-value"

	_, err := UpdateOktaIdpSettings(ctx, r.Encryptor, r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())

	resp, err := UpdateOktaIdpSettings(ctx, r.Encryptor, r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		IssuerBaseUrl:     &issuer,
		OauthClientId:     &clientID,
		OauthClientSecret: &secret,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Settings)
	assert.True(t, resp.Settings.Configured)
	assert.Equal(t, issuer, resp.Settings.IssuerBaseUrl)
	assert.Equal(t, clientID, resp.Settings.OauthClientId)
	assert.True(t, resp.Settings.OauthClientSecretConfigured)
	assert.False(t, resp.Settings.OidcEnabled)
	assert.False(t, resp.Settings.ScimEnabled)

	row, err := models.FindOrganizationOktaIDPByOrganizationID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.NotEmpty(t, row.OAuthClientSecretCiphertext)
	plain, err := r.Encryptor.Decrypt(ctx, row.OAuthClientSecretCiphertext, []byte(oktaOAuthClientSecretCredentialName))
	require.NoError(t, err)
	assert.Equal(t, secret, string(plain))

	issuer2 := "https://dev-example.okta.com/oauth2/custom"
	patch, err := UpdateOktaIdpSettings(ctx, r.Encryptor, r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		IssuerBaseUrl: &issuer2,
	})
	require.NoError(t, err)
	assert.Equal(t, issuer2, patch.Settings.IssuerBaseUrl)
	assert.Equal(t, clientID, patch.Settings.OauthClientId)
}

func Test__RotateOktaScimBearerToken(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := RotateOktaScimBearerToken(r.Organization.ID.String())
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	issuer := "https://dev-example.okta.com/oauth2/default"
	clientID := "cid"
	_, err = UpdateOktaIdpSettings(ctx, r.Encryptor, r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		IssuerBaseUrl: &issuer,
		OauthClientId: &clientID,
	})
	require.NoError(t, err)

	rot, err := RotateOktaScimBearerToken(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotEmpty(t, rot.ScimBearerToken)
	assert.True(t, rot.Settings.ScimBearerTokenConfigured)

	row, err := models.FindOrganizationOktaIDPByOrganizationID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, row.ScimBearerTokenHash)
	assert.Equal(t, crypto.HashToken(rot.ScimBearerToken), *row.ScimBearerTokenHash)
}

func Test__UpdateOktaIdpSettings_InvalidIssuer(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	bad := "http://insecure.example/oauth2/default"
	cid := "x"
	_, err := UpdateOktaIdpSettings(ctx, r.Encryptor, r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		IssuerBaseUrl: &bad,
		OauthClientId: &cid,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
