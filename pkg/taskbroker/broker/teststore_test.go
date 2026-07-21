package broker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/taskbroker/store"
)

func openStore(t *testing.T) *store.PostgresStore {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	return store.NewPostgresStore(database.Conn())
}
