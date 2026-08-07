package secrets

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ResolveSecretDomain(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	otherOrg, err := models.CreateOrganization(support.RandomName("org"), support.RandomName("org-display"))
	require.NoError(t, err)
	otherCanvas, _ := support.CreateCanvas(t, otherOrg.ID, r.User, nil, nil)

	t.Run("unspecified domain type resolves to the organization from context", func(t *testing.T) {
		domainType, domainID, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED, "")
		require.NoError(t, err)
		assert.Equal(t, models.DomainTypeOrganization, domainType)
		assert.Equal(t, r.Organization.ID.String(), domainID)
	})

	t.Run("organization domain type ignores any domain id sent by the client", func(t *testing.T) {
		domainType, domainID, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, uuid.NewString())
		require.NoError(t, err)
		assert.Equal(t, models.DomainTypeOrganization, domainType)
		assert.Equal(t, r.Organization.ID.String(), domainID)
	})

	t.Run("canvas domain type resolves to the requested canvas, when it belongs to the org", func(t *testing.T) {
		domainType, domainID, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_CANVAS, canvas.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.DomainTypeCanvas, domainType)
		assert.Equal(t, canvas.ID.String(), domainID)
	})

	t.Run("canvas domain type is rejected when the canvas belongs to another org", func(t *testing.T) {
		_, _, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_CANVAS, otherCanvas.ID.String())
		require.Error(t, err)
	})

	t.Run("canvas domain type is rejected when the canvas doesn't exist", func(t *testing.T) {
		_, _, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_CANVAS, uuid.NewString())
		require.Error(t, err)
	})

	t.Run("canvas domain type is rejected when the domain id isn't a valid UUID", func(t *testing.T) {
		_, _, err := ResolveSecretDomain(r.Organization.ID.String(), pbAuth.DomainType_DOMAIN_TYPE_CANVAS, "not-a-uuid")
		require.Error(t, err)
	})
}

func Test__SecretNameCollisionAcrossDomains(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	name := support.RandomName("secret")

	data := []byte(`{"key":"value"}`)

	orgSecret, err := models.CreateSecret(name, secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)
	require.NotNil(t, orgSecret)

	//
	// A canvas secret with the same name doesn't collide with the
	// organization secret, since uniqueness is scoped per domain.
	//
	canvasSecret, err := models.CreateSecret(name, secrets.ProviderLocal, r.User.String(), models.DomainTypeCanvas, canvas.ID, data)
	require.NoError(t, err)
	require.NotNil(t, canvasSecret)
	require.NotEqual(t, orgSecret.ID, canvasSecret.ID)
}
