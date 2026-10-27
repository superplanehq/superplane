package models_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func setLegacyBillingPlanTestProduct(t *testing.T) {
	t.Helper()
	db := database.DB(t.Context())
	catalog, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanBusiness)
	require.NoError(t, err)
	originalProduct := catalog.PolarProductID
	if originalProduct != nil {
		originalProduct = new(*originalProduct)
	}
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Model(catalog).Update("polar_product_id", originalProduct).Error)
	})
	require.NoError(t, db.Model(catalog).Update("polar_product_id", "prod_business").Error)
}

func Test__EnsureOrganizationBillingPlanCreatesNone(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Catalog.Key)
	assert.Equal(t, billingmodels.BillingPlanSourceSystem, plan.PlanSource)
	assert.Nil(t, plan.TrialStartedAt)
	assert.Nil(t, plan.TrialEndsAt)
	assert.False(t, plan.AllowsHostedExecution(time.Now()))

	credit, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), credit.GrantMicros)
	require.NotNil(t, credit.WelcomeCreditExpiresAt)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, billingmodels.SignupCreditDays), *credit.WelcomeCreditExpiresAt, time.Minute)
	require.ErrorIs(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID), billingmodels.ErrHostedSubscriptionRequired)

	again, err := billingmodels.EnsureOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.PlanID, again.PlanID)
	assert.Nil(t, again.TrialStartedAt)
	assert.Nil(t, again.TrialEndsAt)

	require.NoError(t, billingmodels.GrantWelcomeCredit(db, r.Organization.ID, r.Account.ID))
	repeated, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, credit.GrantMicros, repeated.GrantMicros)
	stored, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, stored.Plan)
	assert.Nil(t, stored.TrialEndsAt)
}

func Test__InitialPaidPlanAddsConfiguredTrialCredit(t *testing.T) {
	for _, tc := range []struct {
		name       string
		trialCents int64
		polar      bool
	}{
		{name: "pay as you go without trial"},
		{name: "pay as you go with trial", trialCents: 1000},
		{name: "subscription without trial", polar: true},
		{name: "subscription with trial", trialCents: 1200, polar: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restoreInstallationLLMSettings(t)
			r := support.Setup(t)
			db := database.DB(t.Context())
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			plan := billingmodels.BillingPlan{
				Key: "selected_" + uuid.NewString()[:8], Name: "Selected", Category: "paid", Visibility: "public",
				TrialCreditCents: tc.trialCents, CreditFrequency: "none", CreditExpiration: "never",
			}
			product := "prod_" + uuid.NewString()
			if tc.polar {
				plan.PolarProductID = &product
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
			execution := dispatchWorkOrderExecution(t, r)
			recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))
			before := describeCredit(t, db, r.Organization.ID)
			require.Positive(t, before.BilledMicros)
			require.Less(t, before.BilledMicros, billingmodels.CentsToMicros(1200))
			if tc.polar {
				_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
					ID: product, ProductID: product, Status: billingmodels.PolarSubscriptionStatusIncomplete,
				}, nil)
				require.NoError(t, err)
				require.Equal(t, before.WelcomeRemainingMicros, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)
			}

			selectPlan := func() {
				if tc.polar {
					now := time.Now()
					end := now.AddDate(0, 1, 0)
					_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
						ID: product, ProductID: product, Status: billingmodels.PolarSubscriptionStatusActive,
						PeriodStart: &now, PeriodEnd: &end,
					}, nil)
					require.NoError(t, err)
					return
				}
				_, err := billingmodels.ActivateOrganizationBillingPlan(db, r.Organization.ID, plan.Key)
				require.NoError(t, err)
			}
			selectPlan()
			want := before.WelcomeRemainingMicros + billingmodels.CentsToMicros(tc.trialCents)
			require.Equal(t, want, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)
			selectPlan()
			assert.Equal(t, want, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)
		})
	}
}

func Test__InitialPaidPlanTrialCreditIsNotRepeatedAfterReactivation(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	product := "prod_" + uuid.NewString()
	plan := billingmodels.BillingPlan{
		Key: "selected_" + uuid.NewString()[:8], Name: "Selected", Category: "paid", Visibility: "public",
		TrialCreditCents: 1200, CreditFrequency: "none", CreditExpiration: "never", PolarProductID: &product,
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))

	apply := func(status string) *billingmodels.OrganizationBillingPlan {
		now := time.Now()
		end := now.AddDate(0, 1, 0)
		savedResult, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
			ID: product, ProductID: product, Status: status, PeriodStart: &now, PeriodEnd: &end,
		}, nil)
		require.NoError(t, err)
		saved := savedResult.Plan
		return saved
	}

	before := describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros
	apply(billingmodels.PolarSubscriptionStatusActive)
	granted := describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros
	require.Equal(t, before+billingmodels.CentsToMicros(1200), granted)

	canceled := apply(billingmodels.PolarSubscriptionStatusCanceled)
	require.Equal(t, billingmodels.BillingPlanTrial, canceled.Plan)
	require.True(t, canceled.IsOpenTrial(time.Now()))

	apply(billingmodels.PolarSubscriptionStatusActive)
	assert.Equal(t, granted, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)
}

func Test__ApplyPolarSubscriptionReplacesAdminTrialWithPaid(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_admin_trial",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.True(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanBusiness, after.Plan)
	assert.Equal(t, "business_legacy", after.Catalog.Key)
	assert.Equal(t, billingmodels.BillingPlanSourcePolar, after.PlanSource)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionPreservesAdminPlanOnCancel(t *testing.T) {
	for _, catalogKey := range []string{billingmodels.BillingPlanBusinessManual, "superplane_only"} {
		for _, retainedSubscription := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/retained_subscription=%t", catalogKey, retainedSubscription), func(t *testing.T) {
				r := support.Setup(t)
				setLegacyBillingPlanTestProduct(t)
				db := database.DB(t.Context())
				assignedKey := catalogKey
				if catalogKey != billingmodels.BillingPlanBusinessManual {
					assignedKey += "_" + uuid.NewString()[:8]
					catalog := billingmodels.BillingPlan{
						Key: assignedKey, Name: "SuperPlane only", Category: "paid", Visibility: "private",
						CreditFrequency: "none", CreditExpiration: "never",
					}
					require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
					t.Cleanup(func() {
						cleanupDB := database.DB(context.WithoutCancel(t.Context()))
						require.NoError(t, cleanupDB.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
						require.NoError(t, billingmodels.DeleteBillingPlan(cleanupDB, catalog.ID))
					})
				}
				plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, assignedKey, nil)
				require.NoError(t, err)
				if retainedSubscription {
					require.NoError(t, db.Model(plan).Update("polar_subscription_id", "sub_old").Error)
				}

				now := time.Now()
				end := now.AddDate(0, 1, 0)
				result, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
					ProductID: "prod_business", ID: "sub_old", Status: billingmodels.PolarSubscriptionStatusCanceled,
					PeriodStart: &now, PeriodEnd: &end,
				}, nil)
				require.NoError(t, err)
				assert.Equal(t, billingmodels.PolarSubscriptionIgnored, result.Disposition)
				assert.False(t, result.GrantIncluded)

				after, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
				require.NoError(t, err)
				assert.Equal(t, plan.PlanID, after.PlanID)
				assert.Equal(t, assignedKey, after.Catalog.Key)
				assert.Equal(t, billingmodels.BillingPlanSourceAdmin, after.PlanSource)
				assert.True(t, after.IsActiveBusiness())
				assert.True(t, after.AllowsCreditPurchase())
			})
		}
	}
}

func Test__SetAdminOrganizationPlanRejectsPerSeatPlans(t *testing.T) {
	for _, requirePaymentMethod := range []bool{false, true} {
		name := "without payment method requirement"
		if requirePaymentMethod {
			name = "with payment method requirement"
		}
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.Conn()
			catalog := billingmodels.BillingPlan{
				Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
				PerSeat: true, SeatLimit: new(50), RequirePaymentMethod: requirePaymentMethod,
				CreditFrequency: "none", CreditExpiration: "never",
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
			t.Cleanup(func() { require.NoError(t, billingmodels.DeleteBillingPlan(db, catalog.ID)) })
			before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)

			assigned, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, catalog.Key, nil)

			require.EqualError(t, err, "Per-seat plans need a Polar subscription. Use a seat limit override instead.")
			require.Nil(t, assigned)
			after, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func Test__SetAdminOrganizationPlanSerializesCatalogSave(t *testing.T) {
	for _, assignmentFirst := range []bool{false, true} {
		name := "save commits first and assignment rejects the per-seat plan"
		if assignmentFirst {
			name = "assignment commits first and save rejects the seat change"
		}
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			catalog := billingmodels.BillingPlan{
				Key: "admin_race_" + uuid.NewString()[:8], Name: "Admin race", Category: "paid", Visibility: "private",
				PolarProductID: new("product_" + uuid.NewString()), SeatLimit: new(50),
				CreditFrequency: "none", CreditExpiration: "never",
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
			before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			changed := catalog
			changed.PerSeat = true
			tx := db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()

			if assignmentFirst {
				_, err := billingmodels.SetAdminOrganizationPlan(tx, r.Organization.ID, catalog.Key, nil)
				require.NoError(t, err)
				saveError := runAfterCommit(t, db, tx, func(conn *gorm.DB) error {
					return billingmodels.SaveBillingPlan(conn, &changed)
				})
				require.ErrorIs(t, saveError, billingmodels.ErrCatalogPlan)
			} else {
				require.NoError(t, billingmodels.SaveBillingPlan(tx, &changed))
				assignmentError := runAfterCommit(t, db, tx, func(conn *gorm.DB) error {
					_, err := billingmodels.SetAdminOrganizationPlan(conn, r.Organization.ID, catalog.Key, nil)
					return err
				})
				require.EqualError(t, assignmentError, "Per-seat plans need a Polar subscription. Use a seat limit override instead.")
			}

			stored, err := billingmodels.FindBillingPlan(db, catalog.Key)
			require.NoError(t, err)
			require.Equal(t, !assignmentFirst, stored.PerSeat)
			after, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			if assignmentFirst {
				require.Equal(t, catalog.ID, after.PlanID)
				require.Equal(t, billingmodels.BillingPlanSourceAdmin, after.PlanSource)
				return
			}
			require.Equal(t, before, after)
		})
	}
}

func Test__ApplyPolarSubscriptionSerializesCatalogSave(t *testing.T) {
	newLinkedPlan := func(t *testing.T, db *gorm.DB) *billingmodels.BillingPlan {
		t.Helper()
		plan := &billingmodels.BillingPlan{
			Key: "race_" + uuid.NewString()[:8], Name: "Race", Category: "paid", Visibility: "private",
			PolarProductID: new("product_" + uuid.NewString()), PriceCents: 2000,
			CreditFrequency: "none", CreditExpiration: "never",
		}
		require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
		return plan
	}

	t.Run("apply holds the plan lock and save rejects after", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		plan := newLinkedPlan(t, db)

		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()
		_, err := billingmodels.ApplyPolarSubscription(tx, r.Organization.ID, billingmodels.PolarSubscriptionApply{
			ID: "sub_race_apply", ProductID: *plan.PolarProductID, Status: billingmodels.PolarSubscriptionStatusActive,
		}, nil)
		require.NoError(t, err)

		changed := *plan
		changed.PriceCents = 4000
		saveError := runAfterCommit(t, db, tx, func(conn *gorm.DB) error {
			return billingmodels.SaveBillingPlan(conn, &changed)
		})
		require.ErrorIs(t, saveError, billingmodels.ErrCatalogPlan)

		stored, err := billingmodels.FindBillingPlan(db, plan.Key)
		require.NoError(t, err)
		require.Equal(t, int64(2000), stored.PriceCents)
		assigned, err := billingmodels.ListAssignedBillingPlanIDs(db, []uuid.UUID{plan.ID})
		require.NoError(t, err)
		require.True(t, assigned.Contains(plan.ID))
	})

	t.Run("save holds the plan lock and apply re-reads the product", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		plan := newLinkedPlan(t, db)
		productID := *plan.PolarProductID

		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()
		moved := *plan
		moved.PolarProductID = new("moved_" + uuid.NewString())
		require.NoError(t, billingmodels.SaveBillingPlan(tx, &moved))

		applyError := runAfterCommit(t, db, tx, func(conn *gorm.DB) error {
			_, err := billingmodels.ApplyPolarSubscription(conn, r.Organization.ID, billingmodels.PolarSubscriptionApply{
				ID: "sub_race_save", ProductID: productID, Status: billingmodels.PolarSubscriptionStatusActive,
			}, nil)
			return err
		})
		require.ErrorContains(t, applyError, "unknown subscription product")

		assigned, err := billingmodels.ListAssignedBillingPlanIDs(db, []uuid.UUID{plan.ID})
		require.NoError(t, err)
		require.False(t, assigned.Contains(plan.ID))
	})
}

// runAfterCommit requires op on another connection to block on the first
// transaction's locks, then commits the first transaction and returns the
// error op finished with.
func runAfterCommit(t *testing.T, db, first *gorm.DB, op func(*gorm.DB) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pid := make(chan int, 1)
	finished := make(chan struct{})
	var opError error
	go func() {
		defer close(finished)
		opError = db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
			var backendPID int
			if err := conn.Raw("SELECT pg_backend_pid()").Scan(&backendPID).Error; err != nil {
				return err
			}
			pid <- backendPID
			return op(conn.Session(&gorm.Session{NewDB: true}))
		})
	}()
	defer func() { first.Rollback(); <-finished }()
	var backendPID int
	select {
	case backendPID = <-pid:
	case <-finished:
		t.Fatalf("concurrent operation could not start: %v", opError)
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
		t.Fatalf("concurrent operation did not wait for the first transaction: %v", opError)
	default:
	}
	require.NoError(t, first.Commit().Error)
	<-finished
	return opError
}

func Test__SetAdminOrganizationPlanAllowsPolarCustomerWithoutSubscription(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cust_polar"))

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__SetAdminOrganizationPlanRejectsActivePolarSubscription(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	now := time.Now()
	end := now.AddDate(0, 1, 0)
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
		ID:          "sub_paid",
		Status:      billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &now,
		PeriodEnd:   &end,
	}, nil)
	require.NoError(t, err)

	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, nil)
	require.ErrorIs(t, err, billingmodels.ErrPolarManagedBillingPlan)

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourcePolar, plan.PlanSource)
}

func Test__SetAdminOrganizationPlanAllowsEndedPolarSubscription(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	now := time.Now()
	end := now.AddDate(0, 1, 0)
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
		ID:          "sub_ended",
		Status:      billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &now,
		PeriodEnd:   &end,
	}, nil)
	require.NoError(t, err)
	ended := now.Add(-time.Hour)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"trial_ends_at":      ended,
			"current_period_end": ended,
		}).Error)
	_, err = billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
		ID:                "sub_ended",
		Status:            billingmodels.PolarSubscriptionStatusCanceled,
		PeriodStart:       &now,
		PeriodEnd:         &ended,
		CancelAtPeriodEnd: true,
	}, nil)
	require.NoError(t, err)

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__OrganizationBillingIsPolarManaged(t *testing.T) {
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	end := time.Now().Add(time.Hour)
	ended := time.Now().Add(-time.Hour)
	paid := &billingmodels.OrganizationBillingPlan{PolarSubscriptionStatus: billingmodels.PolarSubscriptionStatusActive}
	canceling := &billingmodels.OrganizationBillingPlan{
		PolarSubscriptionStatus: billingmodels.PolarSubscriptionStatusCanceled,
		CancelAtPeriodEnd:       true,
		CurrentPeriodEnd:        &end,
	}
	endedPlan := &billingmodels.OrganizationBillingPlan{
		PlanSource:              billingmodels.BillingPlanSourcePolar,
		PolarSubscriptionStatus: billingmodels.PolarSubscriptionStatusCanceled,
		CancelAtPeriodEnd:       true,
		CurrentPeriodEnd:        &ended,
	}
	pastDueCanceling := &billingmodels.OrganizationBillingPlan{
		PolarSubscriptionStatus: billingmodels.PolarSubscriptionStatusPastDue,
		CancelAtPeriodEnd:       true,
		CurrentPeriodEnd:        &end,
	}
	incomplete := &billingmodels.OrganizationBillingPlan{
		PlanSource:              billingmodels.BillingPlanSourcePolar,
		PolarSubscriptionStatus: billingmodels.PolarSubscriptionStatusIncomplete,
	}
	adminTrial := &billingmodels.OrganizationBillingPlan{PlanSource: billingmodels.BillingPlanSourceAdmin}

	assert.True(t, billingmodels.OrganizationBillingIsPolarManaged(paid))
	assert.True(t, billingmodels.OrganizationBillingIsPolarManaged(canceling))
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(endedPlan))
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(pastDueCanceling))
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(incomplete))
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(adminTrial))
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(nil))

	t.Setenv("POLAR_ACCESS_TOKEN", "")
	assert.False(t, billingmodels.OrganizationBillingIsPolarManaged(paid))
}

func Test__SetAdminOrganizationPlanBusinessHasNoIncludedUsageByDefault(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
	assert.Equal(t, "business_manual", plan.Catalog.Key)
	assert.Equal(t, "Business Manual", plan.Catalog.Name)
	assert.Nil(t, plan.Catalog.PolarProductID)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)
	assert.True(t, plan.CurrentPeriodEnd.After(*plan.CurrentPeriodStart))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Zero(t, summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)

	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, "business_manual", nil)
	require.NoError(t, err)
	summary, err = billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Zero(t, summary.IncludedRemainingMicros)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, "business_legacy", nil)
	require.ErrorIs(t, err, billingmodels.ErrInvalidBillingPlan)
}

func Test__SetAdminOrganizationPlanBusinessGrantsIncludedWhenPeriodWasMissing(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn().Begin()
	require.NoError(t, db.Error)
	defer db.Rollback()
	require.NoError(t, db.Model(&billingmodels.BillingPlan{}).Where("key = ?", billingmodels.BillingPlanBusinessManual).
		Update("credit_amount_cents", billingmodels.DefaultIncludedGrantCents).Error)

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":                 billingmodels.BillingPlanBusiness,
			"plan_source":          billingmodels.BillingPlanSourceAdmin,
			"current_period_start": nil,
			"current_period_end":   nil,
		}).Error)

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SetAdminOrganizationPlanNoneAfterBusinessResubscribe(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.AllowsCreditPurchase())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
}

func Test__SetAdminOrganizationPlanTrialExpiresIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	_, err = billingmodels.AddIncludedLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(500),
		"included-"+r.Organization.ID.String(), time.Now().AddDate(0, 1, 0))
	require.NoError(t, err)

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, nil)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)
}

func Test__SetAdminOrganizationPlanExplicitTrialEndKeepsOpenStart(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)

	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before.TrialStartedAt)
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	adminGrant, err := billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(1100), "ops", nil)
	require.NoError(t, err)
	includedEnd := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	_, err = billingmodels.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(500),
		"order-included-"+r.Organization.ID.String(),
		includedEnd,
	)
	require.NoError(t, err)
	purchased, err := billingmodels.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(2500),
		"order-topup-"+r.Organization.ID.String(),
	)
	require.NoError(t, err)
	adminBefore := reloadGrant(t, db, adminGrant.ID)
	purchasedBefore := reloadGrant(t, db, purchased.ID)

	trialEnd := time.Now().Add(40 * 24 * time.Hour).UTC().Truncate(time.Second)
	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(trialEnd))
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	assert.Equal(t, int64(1), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome))
	assert.Equal(t, welcomeBefore.AmountMicros, welcome.AmountMicros)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(trialEnd))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)

	shorterEnd := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	plan, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, &shorterEnd)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(shorterEnd))
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))
	welcome = requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(shorterEnd))
	assert.Equal(t, welcomeBefore.AmountMicros, welcome.AmountMicros)

	assertGrantUnchanged(t, db, adminBefore)
	assertGrantUnchanged(t, db, purchasedBefore)
	assert.Equal(t, billingmodels.CentsToMicros(500), reloadIncludedAmount(t, db, r.Organization.ID))
}

func Test__SetAdminOrganizationPlanReopensExpiredTrialCredit(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)

	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before.TrialStartedAt)
	ended := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Update("trial_ends_at", ended).Error)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", ended).Error)

	trialEnd := time.Now().Add(21 * 24 * time.Hour).UTC().Truncate(time.Second)
	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.After(*before.TrialStartedAt))
	assert.WithinDuration(t, time.Now(), *plan.TrialStartedAt, 5*time.Second)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(trialEnd))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	assert.Equal(t, int64(1), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome))
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(trialEnd))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)
}

func Test__SetAdminOrganizationPlanCustomTrialSyncsWelcomeExpiry(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	catalog := billingmodels.BillingPlan{
		Key:              "custom_trial",
		Name:             "Custom trial",
		Category:         "trial",
		Visibility:       "private",
		CreditFrequency:  "none",
		CreditExpiration: "never",
		TrialLengthDays:  30,
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	t.Cleanup(func() {
		_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
		require.NoError(t, err)
		require.NoError(t, billingmodels.DeleteBillingPlan(db, catalog.ID))
	})

	ended := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", ended).Error)

	trialEnd := time.Now().Add(21 * 24 * time.Hour).UTC().Truncate(time.Second)
	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, catalog.Key, &trialEnd)
	require.NoError(t, err)
	assert.Equal(t, catalog.Key, plan.Plan)
	assert.Equal(t, catalog.ID, plan.PlanID)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(trialEnd))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(trialEnd))
}

func Test__SetAdminOrganizationPlanTrialDoesNotCreateWelcomeGrant(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)

	trialEnd := time.Now().Add(10 * 24 * time.Hour).UTC().Truncate(time.Second)
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome))
}

func Test__SetAdminOrganizationPlanNilTrialEndKeepsFourteenDayDefault(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)

	plan, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanTrial, nil)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialEndsAt)
	assert.WithinDuration(t, time.Now().Add(billingmodels.DefaultWelcomeGrantTTL), *plan.TrialEndsAt, 5*time.Second)
	require.NotNil(t, plan.TrialStartedAt)
	require.NotNil(t, before.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	require.NotNil(t, welcomeBefore.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))
}

func Test__SetAdminOrganizationPlanBusinessAndNoneClearTrialKeepWelcomeExpiry(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	planBefore, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	ignoredEnd := time.Now().Add(60 * 24 * time.Hour).UTC().Truncate(time.Second)

	business, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, &ignoredEnd)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, business.Plan)
	require.NotNil(t, planBefore.TrialEndsAt)
	assert.Nil(t, business.TrialStartedAt)
	assert.Nil(t, business.TrialEndsAt)
	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))

	none, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, &ignoredEnd)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, none.Plan)
	assert.Nil(t, none.TrialStartedAt)
	assert.Nil(t, none.TrialEndsAt)
	welcome = requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))
}

func requireWelcomeGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) billingmodels.OrganizationLLMCreditGrant {
	t.Helper()
	var grant billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, billingmodels.LLMCreditGrantKindWelcome).First(&grant).Error)
	return grant
}

func countGrants(t *testing.T, db *gorm.DB, orgID uuid.UUID, kind string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", orgID, kind).
		Count(&count).Error)
	return count
}

func reloadGrant(t *testing.T, db *gorm.DB, id uuid.UUID) billingmodels.OrganizationLLMCreditGrant {
	t.Helper()
	var grant billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("id = ?", id).First(&grant).Error)
	return grant
}

func reloadIncludedAmount(t *testing.T, db *gorm.DB, orgID uuid.UUID) int64 {
	t.Helper()
	var grant billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, billingmodels.LLMCreditGrantKindIncluded).First(&grant).Error)
	return grant.AmountMicros
}

func assertGrantUnchanged(t *testing.T, db *gorm.DB, before billingmodels.OrganizationLLMCreditGrant) {
	t.Helper()
	var after billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("id = ?", before.ID).First(&after).Error)
	assert.Equal(t, before.Kind, after.Kind)
	assert.Equal(t, before.AmountMicros, after.AmountMicros)
	if before.ExpiresAt == nil {
		assert.Nil(t, after.ExpiresAt)
		return
	}
	require.NotNil(t, after.ExpiresAt)
	assert.True(t, after.ExpiresAt.Equal(*before.ExpiresAt))
}

func Test__ApplyPolarSubscriptionCancelRestoresOpenTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_restore",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_restore",
			Status:      billingmodels.PolarSubscriptionStatusCanceled,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanTrial, after.Plan)
	assert.True(t, after.IsOpenTrial(time.Now()))
	assert.False(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionCancelEndsPlanAfterTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_lapsed",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)

	ended := now.Add(-time.Hour)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Update("trial_ends_at", ended).Error)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_lapsed",
			Status:      billingmodels.PolarSubscriptionStatusCanceled,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanNone, after.Plan)
	assert.False(t, after.IsOpenTrial(time.Now()))
	assert.False(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionIncompleteKeepsTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_incomplete",
			Status:      billingmodels.PolarSubscriptionStatusIncomplete,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanTrial, after.Plan)
	assert.True(t, after.IsOpenTrial(time.Now()))
	assert.Equal(t, billingmodels.PolarSubscriptionStatusIncomplete, after.PolarSubscriptionStatus)
}

func Test__BillingEnabledDefaultsOn(t *testing.T) {
	t.Setenv(billingmodels.BillingEnabledEnv, "")
	assert.True(t, billingmodels.BillingEnabled())

	for _, value := range []string{"yes", "true", " YES "} {
		t.Setenv(billingmodels.BillingEnabledEnv, value)
		assert.True(t, billingmodels.BillingEnabled(), value)
	}
	for _, value := range []string{"no", "false", " NO "} {
		t.Setenv(billingmodels.BillingEnabledEnv, value)
		assert.False(t, billingmodels.BillingEnabled(), value)
	}
}

func Test__CreateOrganizationSkipsTrialWhenBillingDisabled(t *testing.T) {
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	support.Setup(t)

	org, err := models.CreateOrganization(support.RandomName("no-billing"), "")
	require.NoError(t, err)

	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), org.ID)
	require.NoError(t, err)
	assert.Nil(t, plan)
}

func Test__AssertHostedRunAllowedSkipsLimitsWhenBillingDisabled(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":          billingmodels.BillingPlanNone,
			"plan_source":   billingmodels.BillingPlanSourcePolar,
			"trial_ends_at": time.Now().Add(-time.Hour),
		}).Error)
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	zero := int64(0)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &zero))

	require.NoError(t, billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID))
}

func Test__AssertHostedRunAllowedRequiresSubscriptionWhenPolarConfigured(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":          billingmodels.BillingPlanNone,
			"plan_source":   billingmodels.BillingPlanSourcePolar,
			"trial_ends_at": time.Now().Add(-time.Hour),
		}).Error)

	err := billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
}

func Test__AssertOrganizationPlanAllowsRuns(t *testing.T) {
	for _, polarToken := range []string{"", "oat_test"} {
		t.Run("None plan with leftover credit is blocked, polar token "+polarToken, func(t *testing.T) {
			r := support.Setup(t)
			db := database.Conn()
			t.Setenv("POLAR_ACCESS_TOKEN", polarToken)
			_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
			require.NoError(t, err)
			credit, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
			require.NoError(t, err)
			require.Greater(t, credit.RemainingMicros, int64(0))

			require.ErrorIs(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID), billingmodels.ErrHostedSubscriptionRequired)
			require.ErrorIs(t, billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil), billingmodels.ErrHostedSubscriptionRequired)
		})
	}

	t.Run("open trial is allowed", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		support.AssignOpenSystemTrial(t, r.Organization.ID)

		require.NoError(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID))
		require.NoError(t, billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil))
	})

	t.Run("active paid plan is allowed", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
		require.NoError(t, err)

		require.NoError(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID))
	})

	t.Run("expired trial is blocked", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		expireOrganizationTrial(t, db, r.Organization.ID)

		require.ErrorIs(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID), billingmodels.ErrHostedSubscriptionRequired)
	})

	t.Run("billing disabled allows a None plan", func(t *testing.T) {
		r := support.Setup(t)
		db := database.Conn()
		t.Setenv(billingmodels.BillingEnabledEnv, "no")
		_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
		require.NoError(t, err)

		require.NoError(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, r.Organization.ID))
	})

	t.Run("nil organization is rejected", func(t *testing.T) {
		support.Setup(t)

		err := billingmodels.AssertOrganizationPlanAllowsRuns(database.Conn(), uuid.Nil)
		require.Error(t, err)
		require.NotErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	})
}

func Test__ResolveOrganizationBillingPlanLeavesOpenTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)

	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before)
	require.Equal(t, billingmodels.BillingPlanTrial, before.Plan)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__ResolveOrganizationBillingPlanLapsesExpiredTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	expireOrganizationTrial(t, db, r.Organization.ID)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.AllowsHostedExecution(time.Now()))
	assert.False(t, plan.AllowsCreditPurchase())

	stored, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, stored.Plan)
}

func Test__ResolveOrganizationBillingPlanInsertsNoneWhenMissing(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceSystem, plan.PlanSource)
	assert.False(t, plan.AllowsHostedExecution(time.Now()))

	stored, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, billingmodels.BillingPlanNone, stored.Plan)
}

func Test__ResolveOrganizationBillingPlanLeavesActiveBusiness(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())
	assert.True(t, plan.AllowsCreditPurchase())
}

func Test__ResolveOrganizationBillingPlanExpiresIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()

	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	expireOrganizationTrial(t, db, r.Organization.ID)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
}

func Test__AssertHostedRunAllowedRejectsExpiredTrialWithTopup(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	expireOrganizationTrial(t, db, r.Organization.ID)

	_, err := billingmodels.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(2500),
		uuid.NewString(),
	)
	require.NoError(t, err)

	err = billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)

	stored, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, stored.Plan)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Greater(t, summary.PurchasedRemainingMicros, int64(0))
}

func Test__ApplyPolarSubscriptionCancelAtPeriodEndKeepsBusiness(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_ending",
			Status:            billingmodels.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.True(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanBusiness, after.Plan)
	assert.True(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
	assert.True(t, after.AllowsCreditPurchase())
}

func Test__ApplyPolarSubscriptionCanceledKeepsBusinessUntilPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_cancel_flag",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_cancel_flag",
			Status:            billingmodels.PolarSubscriptionStatusCanceled,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanBusiness, after.Plan)
	assert.True(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionResumeClearsCancelAtPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_keep",
			Status:            billingmodels.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
		nil,
	)
	require.NoError(t, err)

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_keep",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.Equal(t, billingmodels.BillingPlanBusiness, after.Plan)
	assert.False(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionIgnoresStaleSnapshot(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	older := now.Add(-time.Minute)
	newer := now

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_stale",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &older,
		},
		nil,
	)
	require.NoError(t, err)

	resumedResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_stale",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &newer,
		},
		nil,
	)
	require.NoError(t, err)
	resumed := resumedResult.Plan
	require.False(t, resumed.CancelAtPeriodEnd)
	require.NotNil(t, resumed.PolarModifiedAt)
	assert.True(t, resumed.PolarModifiedAt.Equal(newer))

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_stale",
			Status:            billingmodels.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
			ModifiedAt:        &older,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.False(t, after.CancelAtPeriodEnd)
	assert.Equal(t, billingmodels.BillingPlanBusiness, after.Plan)
	require.NotNil(t, after.PolarModifiedAt)
	assert.True(t, after.PolarModifiedAt.Equal(newer))
}

func Test__ApplyPolarSubscriptionTimestampLessSnapshotKeepsProviderOrder(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	provider := now.Add(-30 * time.Second)
	laterWebhook := now.Add(-10 * time.Second)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_order",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &provider,
		},
		nil,
	)
	require.NoError(t, err)

	canceledResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_order",
			Status:            billingmodels.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
		nil,
	)
	require.NoError(t, err)
	canceled := canceledResult.Plan
	assert.True(t, canceled.CancelAtPeriodEnd)
	require.NotNil(t, canceled.PolarModifiedAt)
	assert.True(t, canceled.PolarModifiedAt.Equal(provider))

	afterResult, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_order",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &laterWebhook,
		},
		nil,
	)
	require.NoError(t, err)
	after, grantIncluded := afterResult.Plan, afterResult.GrantIncluded
	assert.False(t, grantIncluded)
	assert.False(t, after.CancelAtPeriodEnd)
	require.NotNil(t, after.PolarModifiedAt)
	assert.True(t, after.PolarModifiedAt.Equal(laterWebhook))
}

func Test__ResolveOrganizationBillingPlanLapsesEndedCancelAtPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, err := billingmodels.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:                "sub_ended",
			Status:            billingmodels.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"current_period_end": now.Add(-time.Hour),
			"trial_ends_at":      now.Add(-2 * time.Hour),
		}).Error)

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.AllowsCreditPurchase())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
}

func expireOrganizationTrial(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", orgID).
		Updates(map[string]any{
			"plan":          billingmodels.BillingPlanTrial,
			"plan_id":       db.Model(&billingmodels.BillingPlan{}).Select("id").Where("key = ?", "trial"),
			"trial_ends_at": time.Now().Add(-time.Hour),
		}).Error)
}

func Test__OrganizationBillingPlanHasLivePolarSubscription(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	subscriptionID := "sub_live"
	blank := "  "

	for _, tc := range []struct {
		name string
		plan billingmodels.OrganizationBillingPlan
		live bool
	}{
		{name: "active", plan: polarPlan(&subscriptionID, "active", false, nil), live: true},
		{name: "trialing", plan: polarPlan(&subscriptionID, "trialing", false, nil), live: true},
		{name: "past due", plan: polarPlan(&subscriptionID, "past_due", false, nil), live: true},
		{name: "paused", plan: polarPlan(&subscriptionID, "paused", false, nil), live: true},
		{name: "status with different case and spaces", plan: polarPlan(&subscriptionID, " Active ", false, nil), live: true},
		{name: "canceled", plan: polarPlan(&subscriptionID, "canceled", false, nil)},
		{name: "revoked", plan: polarPlan(&subscriptionID, "revoked", false, nil)},
		{name: "unpaid", plan: polarPlan(&subscriptionID, "unpaid", false, nil)},
		{name: "incomplete", plan: polarPlan(&subscriptionID, "incomplete", false, nil)},
		{name: "empty status", plan: polarPlan(&subscriptionID, "", false, nil)},
		{name: "scheduled cancellation with period still open", plan: polarPlan(&subscriptionID, "active", true, &future), live: true},
		{name: "canceled with paid period still open", plan: polarPlan(&subscriptionID, "canceled", true, &future), live: true},
		{name: "active scheduled cancellation with period ended", plan: polarPlan(&subscriptionID, "active", true, &past), live: true},
		{name: "active scheduled cancellation without period end", plan: polarPlan(&subscriptionID, "active", true, nil), live: true},
		{name: "canceled after scheduled cancellation ended", plan: polarPlan(&subscriptionID, "canceled", true, &past)},
		{name: "missing subscription ID", plan: polarPlan(nil, "active", false, nil)},
		{name: "blank subscription ID", plan: polarPlan(&blank, "active", false, nil)},
		{
			name: "admin plan with active status",
			plan: billingmodels.OrganizationBillingPlan{
				PlanSource: billingmodels.BillingPlanSourceAdmin, PolarSubscriptionID: &subscriptionID, PolarSubscriptionStatus: "active",
			},
		},
		{
			name: "system plan with active status",
			plan: billingmodels.OrganizationBillingPlan{
				PlanSource: billingmodels.BillingPlanSourceSystem, PolarSubscriptionID: &subscriptionID, PolarSubscriptionStatus: "active",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.live, tc.plan.HasLivePolarSubscription(now))
		})
	}

	t.Run("nil plan", func(t *testing.T) {
		var plan *billingmodels.OrganizationBillingPlan
		assert.False(t, plan.HasLivePolarSubscription(now))
	})
}

func Test__OrganizationBillingPlanMayBeBillable(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	subscriptionID := "sub_billable"
	blank := "  "

	for _, tc := range []struct {
		name     string
		plan     billingmodels.OrganizationBillingPlan
		billable bool
	}{
		{name: "active", plan: polarPlan(&subscriptionID, "active", false, nil), billable: true},
		{name: "trialing", plan: polarPlan(&subscriptionID, "trialing", false, nil), billable: true},
		{name: "past due", plan: polarPlan(&subscriptionID, "past_due", false, nil), billable: true},
		{name: "paused", plan: polarPlan(&subscriptionID, "paused", false, nil), billable: true},
		{name: "unpaid", plan: polarPlan(&subscriptionID, "unpaid", false, nil), billable: true},
		{name: "incomplete", plan: polarPlan(&subscriptionID, "incomplete", false, nil), billable: true},
		{name: "empty status", plan: polarPlan(&subscriptionID, "", false, nil), billable: true},
		{name: "unknown status", plan: polarPlan(&subscriptionID, "suspended", false, nil), billable: true},
		{name: "active scheduled cancellation with period ended", plan: polarPlan(&subscriptionID, "active", true, &past), billable: true},
		{name: "canceled with paid period still open", plan: polarPlan(&subscriptionID, "canceled", true, &future), billable: true},
		{name: "canceled", plan: polarPlan(&subscriptionID, "canceled", false, nil)},
		{name: "canceled after scheduled cancellation ended", plan: polarPlan(&subscriptionID, "canceled", true, &past)},
		{name: "revoked", plan: polarPlan(&subscriptionID, " Revoked ", false, nil)},
		{name: "incomplete expired", plan: polarPlan(&subscriptionID, "incomplete_expired", false, nil)},
		{name: "missing subscription ID", plan: polarPlan(nil, "active", false, nil)},
		{name: "blank subscription ID", plan: polarPlan(&blank, "active", false, nil)},
		{
			name: "admin plan with active status",
			plan: billingmodels.OrganizationBillingPlan{
				PlanSource: billingmodels.BillingPlanSourceAdmin, PolarSubscriptionID: &subscriptionID, PolarSubscriptionStatus: "active",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.billable, tc.plan.MayBeBillable(now))
		})
	}

	t.Run("nil plan", func(t *testing.T) {
		var plan *billingmodels.OrganizationBillingPlan
		assert.False(t, plan.MayBeBillable(now))
	})
}

func polarPlan(subscriptionID *string, status string, cancelAtPeriodEnd bool, periodEnd *time.Time) billingmodels.OrganizationBillingPlan {
	return billingmodels.OrganizationBillingPlan{
		PlanSource:              billingmodels.BillingPlanSourcePolar,
		PolarSubscriptionID:     subscriptionID,
		PolarSubscriptionStatus: status,
		CancelAtPeriodEnd:       cancelAtPeriodEnd,
		CurrentPeriodEnd:        periodEnd,
	}
}

func Test__ListSeatRenewalSyncPlansFindsSeatBasedPlansWithAChangeableSubscription(t *testing.T) {
	support.Setup(t)
	db := database.Conn()
	save := func(perSeat bool) *billingmodels.BillingPlan {
		catalog := billingmodels.BillingPlan{
			Key: "sync_" + uuid.NewString()[:8], Name: "Sync", Category: "paid", Visibility: "private", PerSeat: perSeat,
			CreditFrequency: "none", CreditExpiration: "never",
		}
		if perSeat {
			catalog.SeatLimit = new(50)
		}
		require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
		return &catalog
	}
	perSeat, flat := save(true), save(false)
	now := time.Now().UTC()

	type row struct {
		catalog      *billingmodels.BillingPlan
		status       string
		source       string
		subscription *string
		seats        *int
		renewsIn     *time.Duration
		deleted      bool
		cancelAtEnd  bool
	}
	hours := func(n int) *time.Duration { return new(time.Duration(n) * time.Hour) }
	subscription := func() *string { return new("sub_" + uuid.NewString()) }
	polarSource := billingmodels.BillingPlanSourcePolar
	rows := map[string]row{
		// Included, ordered by renewal. A null renewal comes last.
		"active":                    {catalog: perSeat, status: "active", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(4)},
		"canceling":                 {catalog: perSeat, status: "active", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(1), cancelAtEnd: true},
		"past due":                  {catalog: perSeat, status: "past_due", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(2)},
		"unpaid":                    {catalog: perSeat, status: "unpaid", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(3)},
		"paused":                    {catalog: perSeat, status: "paused", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(5)},
		"lapsed with purchased":     {catalog: flat, status: "past_due", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(6)},
		"lapsed without renewal":    {catalog: flat, status: "unpaid", source: polarSource, subscription: subscription(), seats: new(3)},
		"per seat without purchase": {catalog: perSeat, status: "active", source: polarSource, subscription: subscription(), renewsIn: hours(7)},
		// Excluded.
		"canceled":                    {catalog: perSeat, status: "canceled", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(1)},
		"revoked":                     {catalog: perSeat, status: "revoked", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(1)},
		"admin plan":                  {catalog: perSeat, status: "active", source: billingmodels.BillingPlanSourceAdmin, subscription: subscription(), seats: new(3), renewsIn: hours(1)},
		"deleted organization":        {catalog: perSeat, status: "active", source: polarSource, subscription: subscription(), seats: new(3), renewsIn: hours(1), deleted: true},
		"empty subscription":          {catalog: perSeat, status: "active", source: polarSource, subscription: new(""), seats: new(3), renewsIn: hours(1)},
		"no subscription":             {catalog: perSeat, status: "active", source: polarSource, seats: new(3), renewsIn: hours(1)},
		"not seat based and no seats": {catalog: flat, status: "past_due", source: polarSource, subscription: subscription(), renewsIn: hours(1)},
	}
	ids := map[uuid.UUID]string{}
	for name, tc := range rows {
		org, err := models.CreateOrganizationInTransaction(db, "Org "+uuid.NewString()[:8], "")
		require.NoError(t, err)
		_, err = billingmodels.EnsureOrganizationBillingPlan(db, org.ID)
		require.NoError(t, err)
		var periodEnd *time.Time
		if tc.renewsIn != nil {
			periodEnd = new(now.Add(*tc.renewsIn))
		}
		require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", org.ID).Updates(map[string]any{
			"plan_id": tc.catalog.ID, "plan": tc.catalog.Key, "plan_source": tc.source,
			"polar_subscription_id": tc.subscription, "polar_subscription_status": tc.status,
			"purchased_seats": tc.seats, "current_period_end": periodEnd, "cancel_at_period_end": tc.cancelAtEnd,
		}).Error)
		if tc.deleted {
			require.NoError(t, db.Delete(org).Error)
		}
		ids[org.ID] = name
	}

	plans, err := billingmodels.ListSeatRenewalSyncPlans(db)

	require.NoError(t, err)
	var listed []string
	for _, plan := range plans {
		if name, ok := ids[plan.OrganizationID]; ok {
			listed = append(listed, name)
		}
	}
	assert.Equal(t, []string{
		"canceling", "past due", "unpaid", "active", "paused", "lapsed with purchased", "per seat without purchase", "lapsed without renewal",
	}, listed)
}

func Test__OrganizationSoftDeleteRequiresTerminationProof(t *testing.T) {
	applySubscription := func(t *testing.T, orgID uuid.UUID, cancelAtPeriodEnd bool, periodEnd time.Time) {
		t.Helper()
		start := periodEnd.AddDate(0, -1, 0)
		_, err := billingmodels.ApplyPolarSubscription(database.Conn(), orgID, billingmodels.PolarSubscriptionApply{
			ID: "sub_soft_delete", ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &start, PeriodEnd: &periodEnd, CancelAtPeriodEnd: cancelAtPeriodEnd,
		}, nil)
		require.NoError(t, err)
	}
	requireOrganizationKept := func(t *testing.T, orgID uuid.UUID) {
		t.Helper()
		organization, err := models.FindOrganizationByID(orgID.String())
		require.NoError(t, err)
		assert.False(t, organization.DeletedAt.Valid)
	}

	for _, tc := range []struct {
		name              string
		cancelAtPeriodEnd bool
		periodEnd         time.Time
	}{
		{name: "live subscription", periodEnd: time.Now().AddDate(0, 1, 0)},
		{name: "elapsed scheduled cancellation", cancelAtPeriodEnd: true, periodEnd: time.Now().Add(-time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			setLegacyBillingPlanTestProduct(t)
			db := database.DB(t.Context())
			applySubscription(t, r.Organization.ID, tc.cancelAtPeriodEnd, tc.periodEnd)

			require.ErrorIs(t, r.Organization.SoftDelete(db, nil), billingmodels.ErrOrganizationSubscriptionUnverified)
			require.ErrorIs(t, models.SoftDeleteOrganizationInTransaction(db, r.Organization.ID.String()), billingmodels.ErrOrganizationSubscriptionLive)

			requireOrganizationKept(t, r.Organization.ID)
		})
	}

	t.Run("organization without a subscription is deleted", func(t *testing.T) {
		r := support.Setup(t)
		db := database.DB(t.Context())

		require.NoError(t, models.SoftDeleteOrganizationInTransaction(db, r.Organization.ID.String()))

		_, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
