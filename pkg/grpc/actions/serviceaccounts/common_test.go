package serviceaccounts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestSerializeServiceAccount_WithCreator(t *testing.T) {
	orgID := uuid.New()
	saID := uuid.New()
	creatorID := uuid.New()
	email := "creator@example.com"
	desc := "A bot"

	sa := &models.User{
		ID:             saID,
		OrganizationID: orgID,
		Name:           "my-bot",
		Type:           models.UserTypeServiceAccount,
		Description:    &desc,
		CreatedBy:      &creatorID,
		TokenHash:      "hash",
		CreatedAt:      time.Now().Add(-time.Hour),
		UpdatedAt:      time.Now(),
	}

	creator := &models.User{
		ID:             creatorID,
		OrganizationID: orgID,
		Name:           "Pat Example",
		Email:          &email,
		Type:           models.UserTypeHuman,
	}

	out := serializeServiceAccount(sa, creator)
	require.Equal(t, saID.String(), out.Id)
	require.Equal(t, "my-bot", out.Name)
	require.Equal(t, desc, out.Description)
	require.Equal(t, orgID.String(), out.OrganizationId)
	require.Equal(t, creatorID.String(), out.CreatedBy)
	require.True(t, out.HasToken)
	require.Equal(t, "Pat Example", out.CreatedByName)
	require.Equal(t, email, out.CreatedByEmail)
}

func TestSerializeServiceAccount_NoCreator(t *testing.T) {
	orgID := uuid.New()
	saID := uuid.New()
	creatorID := uuid.New()

	sa := &models.User{
		ID:             saID,
		OrganizationID: orgID,
		Name:           "orphan-bot",
		Type:           models.UserTypeServiceAccount,
		CreatedBy:      &creatorID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	out := serializeServiceAccount(sa, nil)
	require.Equal(t, creatorID.String(), out.CreatedBy)
	require.Empty(t, out.CreatedByName)
	require.Empty(t, out.CreatedByEmail)
}
