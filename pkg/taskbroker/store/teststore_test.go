package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	taskstore "github.com/superplanehq/superplane/pkg/taskbroker/store"
)

func openStore(t *testing.T) *taskstore.PostgresStore {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	return taskstore.NewPostgresStore(database.Conn())
}
