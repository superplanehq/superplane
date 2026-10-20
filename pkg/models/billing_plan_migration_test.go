package models_test

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCatalogMigrationPreservesLegacyBilling(t *testing.T) {
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "catalog_test_" + uuid.NewString()[:8]
	require.NoError(t, tx.Exec(`CREATE SCHEMA `+schema).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO `+schema+`, public`).Error)
	require.NoError(t, tx.Exec(`
 CREATE TABLE organizations(id uuid PRIMARY KEY);
 CREATE TABLE installation_llm_settings(id integer PRIMARY KEY,welcome_grant_cents bigint);
 INSERT INTO installation_llm_settings VALUES(1,2300);
 CREATE TABLE organization_billing_plans(
 organization_id uuid PRIMARY KEY,plan text CONSTRAINT organization_billing_plans_plan CHECK(plan IN ('trial','none','business')),
 plan_source text,polar_subscription_id text,polar_subscription_status text,
 current_period_start timestamptz,current_period_end timestamptz,trial_started_at timestamptz,trial_ends_at timestamptz);
 CREATE TABLE organization_llm_credit_grants(kind text CONSTRAINT organization_llm_credit_grants_kind CHECK(kind IN ('included','welcome')),
 amount_micros bigint CONSTRAINT organization_llm_credit_grants_amount_sign CHECK(amount_micros>0));
 INSERT INTO organization_llm_credit_grants VALUES ('welcome',120000),('included',500000);
 `).Error)
	now := time.Now().UTC().Truncate(time.Second)
	for _, entry := range []struct{ plan, source string }{{"trial", "system"}, {"none", "system"}, {"business", "polar"}, {"business", "admin"}} {
		id := uuid.New()
		require.NoError(t, tx.Exec(`INSERT INTO organizations VALUES (?)`, id).Error)
		require.NoError(t, tx.Exec(`INSERT INTO organization_billing_plans VALUES(?,?,?,?,?,?,?,?,?)`, id, entry.plan, entry.source, "sub-"+id.String(), "active", now, now.AddDate(0, 1, 0), now, now.AddDate(0, 0, 14)).Error)
	}
	var before, after string
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(o) ORDER BY organization_id)::text FROM organization_billing_plans o`).Scan(&before).Error)
	migration, err := os.ReadFile("../../db/migrations/20261020121125_plan-catalog.up.sql")
	require.NoError(t, err)
	require.NoError(t, tx.Exec(string(migration)).Error)
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(o)-'plan_id' ORDER BY organization_id)::text FROM organization_billing_plans o`).Scan(&after).Error)
	require.JSONEq(t, before, after)
	var mismatches int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM organization_billing_plans o JOIN billing_plans p ON p.id=o.plan_id WHERE p.key<>CASE o.plan WHEN 'business' THEN 'business_legacy' ELSE o.plan END`).Scan(&mismatches).Error)
	require.Zero(t, mismatches)
	var amount int64
	require.NoError(t, tx.Raw(`SELECT sum(amount_micros) FROM organization_llm_credit_grants`).Scan(&amount).Error)
	require.Equal(t, int64(620000), amount)
	require.NoError(t, tx.Raw(`SELECT trial_credit_cents FROM billing_plans WHERE key='trial'`).Scan(&amount).Error)
	require.Equal(t, int64(2300), amount)
}

func TestPricingMigrationPreservesFlatAndSeatPrices(t *testing.T) {
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "pricing_test_" + uuid.NewString()[:8]
	require.NoError(t, tx.Exec(`CREATE SCHEMA `+schema).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO `+schema+`, public`).Error)
	require.NoError(t, tx.Exec(`CREATE TABLE billing_plans(key text, base_price_cents bigint, seat_price_cents bigint);
 INSERT INTO billing_plans VALUES ('free',0,0),('flat',2000,0),('seat',0,3000);`).Error)
	for _, path := range []string{
		"../../db/migrations/20261020183712_simplify-plan-pricing.up.sql",
		"../../db/data_migrations/20261020183817_convert-plan-pricing.up.sql",
	} {
		migration, err := os.ReadFile(path)
		require.NoError(t, err)
		// The test owns the transaction so its temporary schema is rolled back.
		sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "BEGIN;", ""), "COMMIT;", "")
		if strings.Contains(path, "data_migrations") {
			require.NoError(t, tx.Exec(`INSERT INTO billing_plans(key,base_price_cents,seat_price_cents) VALUES ('mixed',1000,2000)`).Error)
			require.NoError(t, tx.SavePoint("mixed_prices").Error)
			require.ErrorContains(t, tx.Exec(sql).Error, "Cannot convert plans")
			require.NoError(t, tx.RollbackTo("mixed_prices").Error)
			require.NoError(t, tx.Exec(`DELETE FROM billing_plans WHERE key='mixed'`).Error)
		}
		require.NoError(t, tx.Exec(sql).Error)
	}
	var plans []struct {
		Key        string
		PriceCents int64
		PerSeat    bool
	}
	require.NoError(t, tx.Table("billing_plans").Order("key").Find(&plans).Error)
	require.Len(t, plans, 3)
	require.Equal(t, "flat", plans[0].Key)
	require.Equal(t, int64(2000), plans[0].PriceCents)
	require.False(t, plans[0].PerSeat)
	require.Zero(t, plans[1].PriceCents)
	require.False(t, plans[1].PerSeat)
	require.Equal(t, int64(3000), plans[2].PriceCents)
	require.True(t, plans[2].PerSeat)
}
