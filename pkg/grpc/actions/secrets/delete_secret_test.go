package secrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test__DeleteSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	local := map[string]string{"test": "test"}
	data, _ := json.Marshal(local)

	_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)

	t.Run("secret does not exist -> error", func(t *testing.T) {
		_, err := DeleteSecret(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), "test2")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("secret is deleted", func(t *testing.T) {
		_, err := DeleteSecret(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), "test")
		require.NoError(t, err)

		_, err = models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, "test")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
