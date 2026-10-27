package models_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
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
	migration, err := os.ReadFile("../../../db/migrations/20261021165513_billing-catalog.up.sql")
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
	for _, path := range []string{
		"../../../db/data_migrations/20261021171448_seed-billing-catalog.up.sql",
		"../../../db/data_migrations/20261021224720_remove-business-manual-price-and-usage.up.sql",
	} {
		migration, err := os.ReadFile(path)
		require.NoError(t, err)
		// The test owns the transaction so its temporary schema is rolled back.
		sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "BEGIN;", ""), "COMMIT;", "")
		require.NoError(t, tx.Exec(sql).Error)
	}
	require.NoError(t, tx.Raw(`SELECT jsonb_agg(to_jsonb(o)-'plan_id' ORDER BY organization_id)::text FROM organization_billing_plans o`).Scan(&after).Error)
	require.JSONEq(t, before, after)
	require.NoError(t, tx.Raw(`SELECT count(*) FROM organization_billing_plans o JOIN billing_plans p ON p.id=o.plan_id
 WHERE p.key<>CASE WHEN o.plan='business' AND o.plan_source='admin' THEN 'business_manual'
 WHEN o.plan='business' THEN 'business_legacy' ELSE o.plan END`).Scan(&mismatches).Error)
	require.Zero(t, mismatches)
	var manual struct {
		Name              string
		PolarProductID    *string
		CreditAmountCents int64
		PriceCents        int64
		PerSeat           bool
	}
	require.NoError(t, tx.Table("billing_plans").Where("key = ?", "business_manual").Take(&manual).Error)
	require.Equal(t, "Business Manual", manual.Name)
	require.Nil(t, manual.PolarProductID)
	require.Zero(t, manual.CreditAmountCents)
	require.Zero(t, manual.PriceCents)
	require.False(t, manual.PerSeat)
	var payAsYouGo struct {
		RequirePaymentMethod bool
		TrialCreditCents     int64
		TrialLengthDays      int
	}
	require.NoError(t, tx.Table("billing_plans").Where("key = ?", "pay_as_you_go").Take(&payAsYouGo).Error)
	require.True(t, payAsYouGo.RequirePaymentMethod)
	require.Equal(t, int64(1000), payAsYouGo.TrialCreditCents)
	require.Equal(t, 14, payAsYouGo.TrialLengthDays)
	require.NoError(t, tx.Raw(`SELECT sum(amount_micros) FROM organization_llm_credit_grants`).Scan(&amount).Error)
	require.Equal(t, int64(620000), amount)
}
