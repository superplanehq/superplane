package organizations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

type paymentAnalyticsRecorder struct {
	mu     sync.Mutex
	events []productanalytics.Event
}

func (r *paymentAnalyticsRecorder) Capture(event productanalytics.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}
func (*paymentAnalyticsRecorder) IdentifyOrganizationGroup(productanalytics.OrganizationGroupIdentity) {
}
func (*paymentAnalyticsRecorder) Close() error  { return nil }
func (*paymentAnalyticsRecorder) Enabled() bool { return true }
func (r *paymentAnalyticsRecorder) snapshot() []productanalytics.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]productanalytics.Event(nil), r.events...)
}
func recordPaymentAnalytics(t *testing.T) *paymentAnalyticsRecorder {
	t.Helper()
	recorder := &paymentAnalyticsRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() { productanalytics.SetCurrent(previous) })
	return recorder
}

func TestBillingReadObservesSavedCardLoss(t *testing.T) {
	card := func(id string, year int) map[string]any {
		return map[string]any{"id": id, "type": "card", "method_metadata": map[string]int{"exp_month": 12, "exp_year": year}}
	}
	now := time.Now().UTC()
	for _, tc := range []struct {
		name, reason   string
		initial, final []map[string]any
	}{
		{"removed", "removed", []map[string]any{card("private_method", now.Year()+1)}, []map[string]any{}},
		{"expired", "expired", []map[string]any{card("private_method", now.Year()+1)}, []map[string]any{card("private_method", now.Year()-1)}},
		{"mixed", "unknown", []map[string]any{card("private_method", now.Year()+1), card("second_method", now.Year()+1)}, []map[string]any{card("private_method", now.Year()-1)}},
		{"invalid metadata", "unknown", []map[string]any{card("private_method", now.Year()+1)}, []map[string]any{{"id": "private_method", "type": "card", "method_metadata": map[string]int{"exp_month": 0}}}},
		{"another usable card", "", []map[string]any{card("private_method", now.Year()+1), card("second_method", now.Year()+1)}, []map[string]any{card("second_method", now.Year()+1)}},
		{"initially empty", "", []map[string]any{}, []map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.DB(t.Context()), r.Organization.ID, "customer"))
			recorder := recordPaymentAnalytics(t)
			items := tc.initial
			status := http.StatusOK
			server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				writeFakeJSON(t, w, map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}})
			})
			polartest.UseServer(t, server)
			read := func() error {
				_, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
				return err
			}
			require.NoError(t, read())
			require.Empty(t, recorder.snapshot(), "first observation establishes a baseline")
			items = tc.final
			status = http.StatusNotFound
			require.NoError(t, read())
			require.Empty(t, recorder.snapshot(), "missing customers cannot prove removal")
			status = http.StatusServiceUnavailable
			require.Error(t, read())
			require.Empty(t, recorder.snapshot())
			status = http.StatusOK
			require.NoError(t, read())
			require.NoError(t, read())
			if tc.reason == "" {
				require.Empty(t, recorder.snapshot())
				return
			}
			events := recorder.snapshot()
			require.Len(t, events, 1)
			event := events[0]
			require.Equal(t, "payment_method_removed", string(event.Name))
			require.Equal(t, tc.reason, event.Properties["reason"])
			require.Equal(t, "polar", event.Properties["provider"])
			require.Equal(t, "availability_check", event.Properties["detection_method"])
			require.Equal(t, productanalytics.ActorSystem, event.ActorType)
			require.Equal(t, productanalytics.Source("polar_api"), event.Source)
			require.Equal(t, r.Organization.ID.String(), event.OrganizationID)
			require.Equal(t, productanalytics.SyntheticDistinctID(r.Organization.ID.String(), productanalytics.ActorSystem, "billing"), event.DistinctID)
			_, err := time.Parse(time.RFC3339Nano, event.Properties["observed_at"].(string))
			require.NoError(t, err)
			encoded, err := json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "private_method")
			require.NotContains(t, string(encoded), "second_method")
			items = tc.initial
			require.NoError(t, read())
			items = tc.final
			require.NoError(t, read())
			events = recorder.snapshot()
			require.Len(t, events, 2)
			require.NotEqual(t, events[0].EventID, events[1].EventID, "a recovered card re-arms loss tracking")
		})
	}
}

func TestBillingReadsSerializeInitialObservationsAndLoss(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.DB(t.Context()), r.Organization.ID, "customer"))
	recorder := recordPaymentAnalytics(t)
	var usable atomic.Bool
	usable.Store(true)
	server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
		items := []map[string]any{}
		if usable.Load() {
			items = append(items, map[string]any{"id": "private_method", "type": "kr_card"})
		}
		writeFakeJSON(t, w, map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}})
	})
	polartest.UseServer(t, server)
	readConcurrently := func() {
		t.Helper()
		results := make(chan error, 8)
		start := make(chan struct{})
		for range 8 {
			go func() {
				<-start
				_, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
				results <- err
			}()
		}
		close(start)
		for range 8 {
			require.NoError(t, <-results)
		}
	}
	readConcurrently()
	require.Empty(t, recorder.snapshot())
	usable.Store(false)
	readConcurrently()
	require.Len(t, recorder.snapshot(), 1)
	// A new analytics client has no memory of the baseline. The database keeps it.
	restarted := recordPaymentAnalytics(t)
	readConcurrently()
	require.Empty(t, restarted.snapshot())
	var stored billingmodels.OrganizationPaymentMethodObservation
	require.NoError(t, database.DB(t.Context()).First(&stored, "organization_id = ?", r.Organization.ID).Error)
	require.False(t, stored.Usable)
	require.NotNil(t, stored.TransitionID)
	require.Equal(t, productanalytics.StableEventID(productanalytics.EventPaymentMethodRemoved, r.Organization.ID.String(), stored.TransitionID.String()), recorder.snapshot()[0].EventID)
}

func TestBillingReadRejectsOutOfOrderSnapshots(t *testing.T) {
	for _, staleUsable := range []bool{true, false} {
		t.Run(fmt.Sprintf("stale usable %t", staleUsable), func(t *testing.T) {
			r := support.Setup(t)
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.DB(t.Context()), r.Organization.ID, "customer"))
			recorder := recordPaymentAnalytics(t)
			blocked, release := make(chan struct{}), make(chan struct{})
			var requests atomic.Int32
			server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
				call := requests.Add(1)
				usable := call == 1 || !staleUsable
				if call == 2 {
					usable = staleUsable
					close(blocked)
					select {
					case <-release:
					case <-req.Context().Done():
						return
					}
				}
				items := []map[string]any{}
				if usable {
					items = append(items, map[string]any{"id": "private_method", "type": "kr_card"})
				}
				writeFakeJSON(t, w, map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}})
			})
			polartest.UseServer(t, server)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			read := func() error {
				_, err := DescribeOrganizationBilling(ctx, r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
				return err
			}
			require.NoError(t, read())
			result := make(chan error, 1)
			go func() { result <- read() }()
			select {
			case <-blocked:
			case <-ctx.Done():
				t.Fatal("older read did not start")
			}
			require.NoError(t, read(), "provider requests must not hold database locks")
			close(release)
			require.NoError(t, <-result)
			require.NoError(t, read())
			var stored billingmodels.OrganizationPaymentMethodObservation
			require.NoError(t, database.DB(ctx).First(&stored, "organization_id = ?", r.Organization.ID).Error)
			require.Equal(t, !staleUsable, stored.Usable)
			if staleUsable {
				require.Len(t, recorder.snapshot(), 1)
			} else {
				require.Empty(t, recorder.snapshot())
			}
		})
	}
}

func TestBillingObservationFailurePreservesResponseAndBaseline(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(db, r.Organization.ID, "customer"))
	recorder := recordPaymentAnalytics(t)
	usable, incomplete := true, false
	server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
		if incomplete && req.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		items := []map[string]any{}
		if usable {
			items = append(items, map[string]any{"id": "private_method", "type": "kr_card"})
		}
		pages := 1
		if incomplete {
			pages = 2
		}
		writeFakeJSON(t, w, map[string]any{"items": items, "pagination": map[string]int{"max_page": pages}})
	})
	polartest.UseServer(t, server)
	read := func() (*pb.DescribeOrganizationBillingResponse, error) {
		return DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	}
	_, err := read()
	require.NoError(t, err)
	var baseline billingmodels.OrganizationPaymentMethodObservation
	require.NoError(t, db.First(&baseline, "organization_id = ?", r.Organization.ID).Error)
	usable, incomplete = false, true
	_, err = read()
	require.Error(t, err)
	require.Empty(t, recorder.snapshot())
	incomplete = false
	name := "reject_observation_" + strings.ReplaceAll(r.Organization.ID.String(), "-", "")
	require.NoError(t, db.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id = '%s'::uuid THEN RAISE EXCEPTION 'observation failure'; END IF; RETURN NEW; END $$`, name, r.Organization.ID)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE ON organization_payment_method_observations FOR EACH ROW EXECUTE FUNCTION %s()`, name, name)).Error)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s() CASCADE", name)).Error)
	})
	response, err := read()
	require.NoError(t, err, "observation failures do not fail the billing response")
	require.False(t, response.HasPaymentMethod)
	require.Empty(t, recorder.snapshot(), "a rolled-back observation cannot emit")
	var unchanged billingmodels.OrganizationPaymentMethodObservation
	require.NoError(t, db.First(&unchanged, "organization_id = ?", r.Organization.ID).Error)
	require.Equal(t, baseline.ReadStartedAt, unchanged.ReadStartedAt)
	require.True(t, unchanged.Usable)
	require.NoError(t, db.Exec(fmt.Sprintf("DROP FUNCTION %s() CASCADE", name)).Error)
	_, err = read()
	require.NoError(t, err)
	require.Len(t, recorder.snapshot(), 1)
	// Hard deletion removes the private baseline through the foreign-key cascade.
	organization, err := models.CreateOrganizationInTransaction(db, support.RandomName("empty"), "")
	require.NoError(t, err)
	now := time.Now()
	_, err = billingmodels.ObserveOrganizationPaymentMethods(db, billingmodels.PaymentMethodObservation{OrganizationID: organization.ID, ReadStartedAt: now, ObservedAt: now})
	require.NoError(t, err)
	require.NoError(t, db.Unscoped().Delete(organization).Error)
	var count int64
	require.NoError(t, db.Model(&billingmodels.OrganizationPaymentMethodObservation{}).Where("organization_id = ?", organization.ID).Count(&count).Error)
	require.Zero(t, count)
}
