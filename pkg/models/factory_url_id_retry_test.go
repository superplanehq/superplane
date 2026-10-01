package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
)

func TestCreateFactory_RetriesURLIDCollisionInTransaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, _, _ := setupFactoryWithUser(t, "url-id-retry")
	db := database.DB(t.Context())

	first, err := CreateFactory(db, org.ID, "First workspace", "", "AA")
	require.NoError(t, err)
	require.Regexp(t, `^[a-z0-9]{8}$`, first.URLID)

	retryID := "zzzzzzzz"
	if first.URLID == retryID {
		retryID = "yyyyyyyy"
	}

	calls := 0
	orig := newFactoryURLID
	t.Cleanup(func() { newFactoryURLID = orig })
	newFactoryURLID = func() (string, error) {
		calls++
		if calls == 1 {
			return first.URLID, nil
		}
		return retryID, nil
	}

	var second *Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		created, createErr := CreateFactory(tx, org.ID, "Second workspace", "", "BB")
		if createErr != nil {
			return createErr
		}
		second = created
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, retryID, second.URLID)
	assert.Equal(t, 2, calls)
}
