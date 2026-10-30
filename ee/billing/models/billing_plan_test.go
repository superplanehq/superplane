package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestCatalogPlanAssignmentAndProtection(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	zero := 0
	plan := billingmodels.BillingPlan{Key: "custom_" + uuid.NewString()[:8], Name: "Custom", Category: "paid", Visibility: "private", MarkupBPS: &zero, CreditAmountCents: 1234, CreditFrequency: "once", CreditExpiration: "after_days"}
	require.Error(t, billingmodels.SaveBillingPlan(db, &plan))
	days := 30
	plan.CreditValidityDays = &days
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	assigned, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.True(t, assigned.IsActiveBusiness())
	require.Nil(t, assigned.TrialStartedAt)
	require.Nil(t, assigned.TrialEndsAt)
	require.Equal(t, plan.ID, assigned.PlanID)
	markup, err := billingmodels.ResolveOrganizationMarkupBPS(db, r.Organization.ID)
	require.NoError(t, err)
	require.Zero(t, markup)
	err = billingmodels.DeleteBillingPlan(db, plan.ID)
	require.ErrorIs(t, err, billingmodels.ErrCatalogPlan)
	require.ErrorContains(t, err, billingmodels.BillingPlanDeleteReasonAssigned)
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	for _, grant := range grants {
		require.NotEqual(t, billingmodels.LLMCreditGrantKindIncluded, grant.Kind)
	}
	original := plan.Key
	plan.Key = "changed"
	require.Error(t, billingmodels.SaveBillingPlan(db, &plan))
	plan.Key = original
	product := "product_" + uuid.NewString()
	plan.PolarProductID = &product
	plan.PriceCents = 3000
	plan.PerSeat = true
	plan.SeatLimit = new(50)
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	require.ErrorContains(t, billingmodels.DeleteBillingPlan(db, plan.ID), billingmodels.BillingPlanDeleteReasonPolar)
	stored, err := billingmodels.FindBillingPlan(db, plan.Key)
	require.NoError(t, err)
	require.Equal(t, int64(3000), stored.PriceCents)
	require.True(t, stored.PerSeat)
	plan.PerSeat = false
	require.ErrorIs(t, billingmodels.SaveBillingPlan(db, &plan), billingmodels.ErrCatalogPlan)
	plan.PerSeat = true
	plan.SeatLimit = new(1000)
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	plan.PolarProductID = new("replacement_product")
	require.ErrorIs(t, billingmodels.SaveBillingPlan(db, &plan), billingmodels.ErrCatalogPlan)
	plan.PolarProductID = &product
	plan.PriceCents = 2000
	require.ErrorIs(t, billingmodels.SaveBillingPlan(db, &plan), billingmodels.ErrCatalogPlan)
	manual, err := billingmodels.FindBillingPlan(db, "business_manual")
	require.NoError(t, err)
	require.ErrorIs(t, billingmodels.DeleteBillingPlan(db, manual.ID), billingmodels.ErrCatalogPlan)
	require.ErrorContains(t, billingmodels.DeleteBillingPlan(db, manual.ID), billingmodels.BillingPlanDeleteReasonSystem)
	manual.Category = "trial"
	require.ErrorIs(t, billingmodels.SaveBillingPlan(db, manual), billingmodels.ErrCatalogPlan)
	manual.Category = "paid"
	manual.PolarProductID = &product
	require.ErrorIs(t, billingmodels.SaveBillingPlan(db, manual), billingmodels.ErrCatalogPlan)
}

func TestSaveBillingPlanLinkedProductChanges(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	newLinkedPlan := func(t *testing.T) *billingmodels.BillingPlan {
		t.Helper()
		plan := &billingmodels.BillingPlan{Key: "linked_" + uuid.NewString()[:8], Name: "Linked", Category: "paid", Visibility: "private", PolarProductID: new("product_" + uuid.NewString()), PriceCents: 2000, CreditFrequency: "none", CreditExpiration: "never"}
		require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
		return plan
	}

	t.Run("unassigned plan accepts linked changes", func(t *testing.T) {
		plan := newLinkedPlan(t)

		plan.PolarProductID = new("moved_" + uuid.NewString())
		require.NoError(t, billingmodels.SaveBillingPlan(db, plan))

		plan.PriceCents = 4000
		plan.PerSeat = true
		plan.SeatLimit = new(50)
		require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
		stored, err := billingmodels.FindBillingPlan(db, plan.Key)
		require.NoError(t, err)
		require.Equal(t, *plan.PolarProductID, *stored.PolarProductID)
		require.Equal(t, int64(4000), stored.PriceCents)
		require.True(t, stored.PerSeat)

		plan.PolarProductID = nil
		require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
		stored, err = billingmodels.FindBillingPlan(db, plan.Key)
		require.NoError(t, err)
		require.Nil(t, stored.PolarProductID)
		require.NoError(t, billingmodels.DeleteBillingPlan(db, plan.ID))
	})

	t.Run("business manual never takes a product", func(t *testing.T) {
		manual, err := billingmodels.FindBillingPlan(db, "business_manual")
		require.NoError(t, err)
		manual.PolarProductID = new("product_" + uuid.NewString())
		require.ErrorIs(t, billingmodels.SaveBillingPlan(db, manual), billingmodels.ErrCatalogPlan)
	})

	t.Run("assigned plan rejects linked changes", func(t *testing.T) {
		plan := newLinkedPlan(t)
		_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
		require.NoError(t, err)
		original := *plan

		changes := map[string]func(*billingmodels.BillingPlan){
			"clear product":   func(p *billingmodels.BillingPlan) { p.PolarProductID = nil },
			"other product":   func(p *billingmodels.BillingPlan) { p.PolarProductID = new("other_" + uuid.NewString()) },
			"change price":    func(p *billingmodels.BillingPlan) { p.PriceCents = 4000 },
			"toggle per seat": func(p *billingmodels.BillingPlan) { p.PerSeat = true; p.SeatLimit = new(50) },
		}
		for name, change := range changes {
			candidate := original
			change(&candidate)
			require.ErrorIs(t, billingmodels.SaveBillingPlan(db, &candidate), billingmodels.ErrCatalogPlan, name)
		}
		stored, err := billingmodels.FindBillingPlan(db, plan.Key)
		require.NoError(t, err)
		require.Equal(t, *original.PolarProductID, *stored.PolarProductID)
		require.Equal(t, original.PriceCents, stored.PriceCents)
		require.False(t, stored.PerSeat)
	})
}

func TestPerSeatPlanMaximum(t *testing.T) {
	support.Setup(t)
	db := database.Conn()
	for _, tc := range []struct {
		name    string
		maximum *int
		valid   bool
	}{
		{"missing", nil, false},
		{"zero", new(0), false},
		{"negative", new(-1), false},
		{"above Polar limit", new(1001), false},
		{"minimum", new(1), true},
		{"default", new(50), true},
		{"Polar limit", new(1000), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := billingmodels.BillingPlan{Key: "seats_" + uuid.NewString()[:8], Name: "Seats", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: tc.maximum, CreditAmountCents: 1234, CreditFrequency: "billing_period", CreditExpiration: "never"}
			t.Cleanup(func() { require.NoError(t, db.Where("key = ?", plan.Key).Delete(&billingmodels.BillingPlan{}).Error) })
			err := billingmodels.SaveBillingPlan(db, &plan)
			if !tc.valid {
				require.ErrorIs(t, err, billingmodels.ErrCatalogPlan)
				return
			}
			require.NoError(t, err)
			stored, err := billingmodels.FindBillingPlan(db, plan.Key)
			require.NoError(t, err)
			require.Equal(t, *tc.maximum, *stored.SeatLimit)
			require.Equal(t, int64(1234), stored.CreditAmountCents)
		})
	}
	for _, tc := range []struct {
		name    string
		maximum *int
	}{{"unlimited flat plan", nil}, {"large flat plan", new(1001)}} {
		t.Run(tc.name, func(t *testing.T) {
			plan := billingmodels.BillingPlan{Key: "flat_" + uuid.NewString()[:8], Name: "Flat", Category: "paid", Visibility: "private", SeatLimit: tc.maximum, CreditFrequency: "none", CreditExpiration: "never"}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
			t.Cleanup(func() { require.NoError(t, billingmodels.DeleteBillingPlan(db, plan.ID)) })
		})
	}
}

func TestFeatureDeletionSerializesReferenceWrites(t *testing.T) {
	for _, writer := range []string{"create plan", "update plan", "enabled grant", "disabled grant"} {
		for _, deleteFirst := range []bool{true, false} {
			order := "write first"
			if deleteFirst {
				order = "delete first"
			}
			t.Run(writer+"/"+order, func(t *testing.T) {
				r := support.Setup(t)
				db := database.Conn()
				key := "overlap_" + uuid.NewString()[:8]
				feature := billingmodels.PlanFeatureDefinition{Key: key, Label: "Overlap", Gated: true}
				require.NoError(t, billingmodels.CreatePlanFeatureDefinition(db, &feature))
				plan := billingmodels.BillingPlan{Key: key, Name: "Overlap", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
				if writer == "update plan" {
					require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
				}
				t.Cleanup(func() {
					require.NoError(t, db.Where("feature_key = ?", key).Delete(&billingmodels.OrganizationFeatureGrant{}).Error)
					require.NoError(t, db.Where("key = ?", key).Delete(&billingmodels.BillingPlan{}).Error)
					require.NoError(t, db.Where("key = ?", key).Delete(&billingmodels.PlanFeatureDefinition{}).Error)
				})
				write := func(tx *gorm.DB) error {
					if writer == "create plan" || writer == "update plan" {
						plan.Features = []string{key}
						return billingmodels.SaveBillingPlan(tx, &plan)
					}
					return billingmodels.SetOrganizationFeatureGrant(tx, billingmodels.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: key, Enabled: writer == "enabled grant"})
				}
				remove := func(tx *gorm.DB) error { return billingmodels.DeletePlanFeatureDefinition(tx, key) }
				first, second := write, remove
				if deleteFirst {
					first, second = remove, write
				}
				tx := db.Begin()
				require.NoError(t, tx.Error)
				defer tx.Rollback()
				require.NoError(t, first(tx))

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				pid := make(chan int, 1)
				finished := make(chan struct{})
				var secondError error
				go func() {
					defer close(finished)
					secondError = db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
						var backendPID int
						if err := conn.Raw("SELECT pg_backend_pid()").Scan(&backendPID).Error; err != nil {
							return err
						}
						pid <- backendPID
						return second(conn)
					})
				}()
				defer func() { tx.Rollback(); <-finished }()
				var backendPID int
				select {
				case backendPID = <-pid:
				case <-finished:
					t.Fatalf("reference operation could not start: %v", secondError)
				}
				require.Eventually(t, func() bool {
					select {
					case <-finished:
						return true
					default:
					}
					var blocked bool
					err := db.Raw("SELECT cardinality(pg_blocking_pids(?)) > 0", backendPID).Scan(&blocked).Error
					return err == nil && blocked
				}, 5*time.Second, 10*time.Millisecond)
				select {
				case <-finished:
					t.Fatalf("overlapping operation did not wait for the transaction: %v", secondError)
				default:
				}
				require.NoError(t, tx.Commit().Error)
				<-finished
				if deleteFirst {
					require.ErrorIs(t, secondError, billingmodels.ErrCatalogPlan)
				} else {
					require.NoError(t, secondError)
				}
				require.NoError(t, billingmodels.CreatePlanFeatureDefinition(db, &feature))
				var grants int64
				require.NoError(t, db.Model(&billingmodels.OrganizationFeatureGrant{}).Where("feature_key = ?", key).Count(&grants).Error)
				require.Zero(t, grants)
				var plans []billingmodels.BillingPlan
				require.NoError(t, db.Where("key = ?", key).Find(&plans).Error)
				for _, stored := range plans {
					require.NotContains(t, stored.Features, key)
				}
			})
		}
	}
}
