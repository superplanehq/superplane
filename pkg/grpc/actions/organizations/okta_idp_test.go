package organizations

import (
	"slices"
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

	ssoURL := "https://dev-example.okta.com/app/superplane/exk123/sso/saml"
	issuer := "http://www.okta.com/exk123"
	cert := "-----BEGIN CERTIFICATE-----\nMIICmDCCAYAC\n-----END CERTIFICATE-----"

	_, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())

	resp, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl:         &ssoURL,
		SamlIdpIssuer:         &issuer,
		SamlIdpCertificatePem: &cert,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Settings)
	assert.True(t, resp.Settings.Configured)
	assert.Equal(t, ssoURL, resp.Settings.SamlIdpSsoUrl)
	assert.Equal(t, issuer, resp.Settings.SamlIdpIssuer)
	assert.True(t, resp.Settings.SamlIdpCertificateConfigured)
	assert.False(t, resp.Settings.SamlEnabled)
	assert.False(t, resp.Settings.ScimEnabled)

	row, err := models.FindOrganizationOktaIDPByOrganizationID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.NotEmpty(t, row.SamlIdpCertificatePEM)

	ssoURL2 := "https://dev-example.okta.com/app/superplane/exk456/sso/saml"
	patch, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl: &ssoURL2,
	})
	require.NoError(t, err)
	assert.Equal(t, ssoURL2, patch.Settings.SamlIdpSsoUrl)
	assert.Equal(t, issuer, patch.Settings.SamlIdpIssuer)
}

func Test__RotateOktaScimBearerToken(t *testing.T) {
	r := support.Setup(t)

	_, err := RotateOktaScimBearerToken(r.Organization.ID.String())
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	ssoURL := "https://dev-example.okta.com/app/superplane/exk123/sso/saml"
	issuer := "http://www.okta.com/exk123"
	_, err = UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl: &ssoURL,
		SamlIdpIssuer: &issuer,
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

func Test__UpdateOktaIdpSettings_SAMLRequiresCertificate(t *testing.T) {
	r := support.Setup(t)

	ssoURL := "https://dev-example.okta.com/app/superplane/exk123/sso/saml"
	issuer := "http://www.okta.com/exk123"
	samlOn := true
	_, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl: &ssoURL,
		SamlIdpIssuer: &issuer,
		SamlEnabled:   &samlOn,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func Test__UpdateOktaIdpSettings_SAMLEnablesOrgProvider(t *testing.T) {
	r := support.Setup(t)

	ssoURL := "https://dev-example.okta.com/app/superplane/exk123/sso/saml"
	issuer := "http://www.okta.com/exk123"
	cert := "-----BEGIN CERTIFICATE-----\nMIICmDCCAYAC\n-----END CERTIFICATE-----"
	samlOn := true
	_, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl:         &ssoURL,
		SamlIdpIssuer:         &issuer,
		SamlIdpCertificatePem: &cert,
		SamlEnabled:           &samlOn,
	})
	require.NoError(t, err)

	org, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.True(t, slices.Contains(org.AllowedProviders, models.ProviderOkta))
}

func Test__UpdateOktaIdpSettings_InvalidSSOURL(t *testing.T) {
	r := support.Setup(t)

	bad := "http://insecure.example/app/saml"
	issuer := "http://www.okta.com/exk123"
	_, err := UpdateOktaIdpSettings(r.Organization.ID.String(), &pb.UpdateOktaIdpSettingsRequest{
		SamlIdpSsoUrl: &bad,
		SamlIdpIssuer: &issuer,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
