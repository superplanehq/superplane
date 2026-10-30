package models_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFindRoleMetadataUsesCallerTransaction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	role := models.RoleMetadata{
		RoleName:    "support",
		DomainType:  models.DomainTypeOrganization,
		DomainID:    r.Organization.ID.String(),
		DisplayName: "Support",
	}
	require.NoError(t, tx.Create(&role).Error)

	found, err := models.FindRoleMetadata(tx, role.RoleName, role.DomainType, role.DomainID)
	require.NoError(t, err)
	require.Equal(t, role.ID, found.ID)
	require.Equal(t, role.DisplayName, found.DisplayName)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = models.FindRoleMetadata(tx.WithContext(ctx), role.RoleName, role.DomainType, role.DomainID)
	require.ErrorIs(t, err, context.Canceled)
}
