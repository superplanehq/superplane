package models_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

// The operation key migration must not invent keys, fingerprints, periods, or
// outcomes for purchases that were recorded before it.
func TestSeatPurchaseOperationKeyMigrationKeepsExistingPurchases(t *testing.T) {
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "seat_purchase_test_" + uuid.NewString()[:8]
	require.NoError(t, tx.Exec(`CREATE SCHEMA `+schema).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO `+schema+`, public`).Error)
	require.NoError(t, tx.Exec(`
 CREATE TABLE organization_seat_purchases(
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), organization_id uuid NOT NULL, user_id uuid,
 polar_subscription_id text NOT NULL, from_seats integer NOT NULL, to_seats integer NOT NULL,
 status text NOT NULL, error_code text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now());
 CREATE UNIQUE INDEX uq_organization_seat_purchases_in_flight ON organization_seat_purchases (organization_id)
 WHERE status IN ('pending', 'unknown');
 `).Error)
	orgID := uuid.New()
	for _, status := range []string{"succeeded", "failed", "action_required", "unknown"} {
		require.NoError(t, tx.Exec(`INSERT INTO organization_seat_purchases (organization_id, polar_subscription_id, from_seats, to_seats, status) VALUES (?, 'sub_team', 3, 4, ?)`, orgID, status).Error)
	}
	var before, after string
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(p) ORDER BY id)::text FROM organization_seat_purchases p`).Scan(&before).Error)

	migration, err := os.ReadFile("../../../db/migrations/20261025145014_add-seat-purchase-operation-key.up.sql")
	require.NoError(t, err)
	require.NoError(t, tx.Exec(string(migration)).Error)

	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(p) - 'operation_key' - 'request_fingerprint' - 'period_start' - 'period_end' ORDER BY id)::text FROM organization_seat_purchases p`).Scan(&after).Error)
	require.JSONEq(t, before, after, "status and every other column stay as they were")
	var filled int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM organization_seat_purchases
 WHERE operation_key IS NOT NULL OR request_fingerprint IS NOT NULL OR period_start IS NOT NULL OR period_end IS NOT NULL`).Scan(&filled).Error)
	require.Zero(t, filled)

	// The existing unknown purchase still blocks a new one.
	err = tx.Exec(`SAVEPOINT in_flight`).Error
	require.NoError(t, err)
	require.Error(t, tx.Exec(`INSERT INTO organization_seat_purchases (organization_id, polar_subscription_id, from_seats, to_seats, status) VALUES (?, 'sub_team', 3, 4, 'pending')`, orgID).Error)
	require.NoError(t, tx.Exec(`ROLLBACK TO SAVEPOINT in_flight`).Error)
}
