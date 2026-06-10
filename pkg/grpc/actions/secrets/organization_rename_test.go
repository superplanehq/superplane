package secrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
	secretpb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SecretsAfterOrganizationRename(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}
	secretName := support.RandomName("secret")
	data, err := json.Marshal(map[string]string{"token": "value"})
	require.NoError(t, err)

	_, err = models.CreateSecret(secretName, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)

	_, err = organizations.UpdateOrganization(context.Background(), r.Organization.ID.String(), &organizationpb.Organization{
		Metadata: &organizationpb.Organization_Metadata{
			Name: support.RandomName("renamed-org"),
		},
	})
	require.NoError(t, err)

	listResponse, err := ListSecrets(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Secrets, 1)
	assert.Equal(t, secretName, listResponse.Secrets[0].Metadata.Name)
	assert.Equal(t, secretpb.Secret_PROVIDER_LOCAL, listResponse.Secrets[0].Spec.Provider)
	require.NotNil(t, listResponse.Secrets[0].Spec.Local)
	assert.Equal(t, map[string]string{"token": "***"}, listResponse.Secrets[0].Spec.Local.Data)

	describeResponse, err := DescribeSecret(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secretName)
	require.NoError(t, err)
	require.NotNil(t, describeResponse.Secret)
	assert.Equal(t, secretName, describeResponse.Secret.Metadata.Name)
	require.NotNil(t, describeResponse.Secret.Spec.Local)
	assert.Equal(t, map[string]string{"token": "***"}, describeResponse.Secret.Spec.Local.Data)
}
