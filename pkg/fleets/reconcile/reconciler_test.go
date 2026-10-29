package reconcile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

type fakeAdmin struct {
	fleet             adminclient.Fleet
	capacity          adminclient.Capacity
	describeError     error
	describeCalls     chan time.Time
	activeRunners     []adminclient.Runner
	terminatedRunners []adminclient.Runner
	createResponse    adminclient.CreateRunnerResponse
	createRequests    []adminclient.CreateRunnerRequest
	deletedRunnerIDs  []string
	providerDeleted   []string
	deleteStates      [][]string
	deleteResponse    *adminclient.Runner
	deleteError       error
	capacityWaits     []int
	events            *[]string
}

func (f *fakeAdmin) DescribeFleet(ctx context.Context, _ string) (adminclient.Fleet, error) {
	if f.describeCalls != nil {
		select {
		case f.describeCalls <- time.Now():
		case <-ctx.Done():
			return adminclient.Fleet{}, ctx.Err()
		}
	}
	return f.fleet, f.describeError
}

func (f *fakeAdmin) DescribeRunner(
	_ context.Context,
	_, runnerID string,
) (adminclient.Runner, error) {
	for _, runner := range append(f.activeRunners, f.terminatedRunners...) {
		if runner.ID == runnerID {
			return runner, nil
		}
	}
	return adminclient.Runner{}, &adminclient.HTTPError{StatusCode: http.StatusNotFound}
}

func (f *fakeAdmin) GetFleetCapacity(
	_ context.Context,
	_ string,
	_ string,
	waitSeconds int,
) (adminclient.Capacity, error) {
	f.capacityWaits = append(f.capacityWaits, waitSeconds)
	return f.capacity, nil
}

func (f *fakeAdmin) CreateRunner(
	_ context.Context,
	_ string,
	request adminclient.CreateRunnerRequest,
) (adminclient.CreateRunnerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.events != nil {
		*f.events = append(*f.events, "admin-create")
	}
	return f.createResponse, nil
}

func (f *fakeAdmin) ListRunners(
	_ context.Context,
	_ string,
	states []string,
	limit int,
) ([]adminclient.Runner, error) {
	if len(states) == 1 && states[0] == adminclient.RunnerStateTerminated {
		return limitedRunners(f.terminatedRunners, limit), nil
	}
	return limitedRunners(f.activeRunners, limit), nil
}

func limitedRunners(runners []adminclient.Runner, limit int) []adminclient.Runner {
	if limit <= 0 || len(runners) <= limit {
		return runners
	}
	return runners[:limit]
}

func (f *fakeAdmin) ListRunnersPage(
	_ context.Context,
	_ string,
	request adminclient.ListRunnersRequest,
) (adminclient.RunnerListPage, error) {
	runners := f.terminatedRunners
	if request.AfterID != "" {
		for i, runner := range runners {
			if runner.ID == request.AfterID {
				runners = runners[i+1:]
				break
			}
		}
	}
	return adminclient.RunnerListPage{
		Runners:     limitedRunners(runners, request.Limit),
		HasNextPage: request.Limit > 0 && len(runners) > request.Limit,
	}, nil
}

func (f *fakeAdmin) MarkRunnerProviderDeleted(
	_ context.Context,
	_, runnerID string,
	_ adminclient.ProviderDeletionRequest,
) (adminclient.Runner, error) {
	f.providerDeleted = append(f.providerDeleted, runnerID)
	if f.events != nil {
		*f.events = append(*f.events, "provider-deleted/"+runnerID)
	}
	return adminclient.Runner{ID: runnerID, State: adminclient.RunnerStateTerminated}, nil
}

func (f *fakeAdmin) DeleteRunner(
	_ context.Context,
	_, runnerID string, states []string,
) (adminclient.Runner, error) {
	f.deletedRunnerIDs = append(f.deletedRunnerIDs, runnerID)
	f.deleteStates = append(f.deleteStates, states)
	if f.deleteError != nil {
		return adminclient.Runner{}, f.deleteError
	}
	if f.deleteResponse != nil {
		return *f.deleteResponse, nil
	}
	return adminclient.Runner{ID: runnerID, State: adminclient.RunnerStateTerminated}, nil
}

type fakeArtifactResolver struct {
	requestedVersion string
	requestedOS      string
	requestedArch    string
}

func (f *fakeArtifactResolver) Resolve(
	_ context.Context,
	version, operatingSystem, architecture string,
) (artifact.Artifact, error) {
	f.requestedVersion = version
	f.requestedOS = operatingSystem
	f.requestedArch = architecture
	return artifact.Artifact{
		Version:         version,
		OperatingSystem: operatingSystem,
		Architecture:    architecture,
		URL:             "https://downloads.example/runner",
		SHA256:          "digest",
	}, nil
}

type fakeProvider struct {
	resources           []provider.Resource
	createRequest       provider.CreateRequest
	bootstrap           provider.RunnerBootstrap
	createError         error
	deleteError         error
	deleteErrors        map[string]error
	synchronousDeletion bool
	deleted             []provider.Resource
	events              *[]string
}

func (f *fakeProvider) Name() string {
	return "fake"
}

func (f *fakeProvider) List(context.Context, string) ([]provider.Resource, error) {
	return f.resources, nil
}

func (f *fakeProvider) BuildBootstrap(request provider.RunnerBootstrap) ([]byte, error) {
	f.bootstrap = request
	return []byte("bootstrap"), nil
}

func (f *fakeProvider) Create(
	_ context.Context,
	request provider.CreateRequest,
) (provider.Resource, error) {
	f.createRequest = request
	if f.events != nil {
		*f.events = append(*f.events, "provider-create")
	}
	if f.createError != nil {
		return provider.Resource{}, f.createError
	}
	return provider.Resource{
		ID:       "resource-1",
		RunnerID: request.RunnerID,
		FleetID:  request.FleetID,
	}, nil
}

func (f *fakeProvider) Delete(_ context.Context, resource provider.Resource) error {
	f.deleted = append(f.deleted, resource)
	if f.events != nil {
		*f.events = append(*f.events, "provider-delete/"+resource.ID)
	}
	if f.deleteError != nil {
		return f.deleteError
	}
	if err := f.deleteErrors[resource.ID]; err != nil {
		return err
	}
	if f.synchronousDeletion {
		f.resources = slices.DeleteFunc(f.resources, func(candidate provider.Resource) bool {
			return candidate.ID == resource.ID
		})
	}
	return nil
}

func (f *fakeProvider) DeletionIsSynchronous() bool {
	return f.synchronousDeletion
}

func TestQueuedTaskAddsCapacityWithoutReplacingWarmRunner(t *testing.T) {
	var events []string
	admin := newFakeAdmin()
	admin.events = &events
	admin.capacity.RunnableTasks = 1
	admin.capacity.IdleRunners = 1
	admin.activeRunners = []adminclient.Runner{{
		ID:      "runner-idle",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateIdle,
	}}
	resourceProvider := &fakeProvider{events: &events}
	resolver := &fakeArtifactResolver{}
	reconciler := newTestReconciler(t, admin, resolver, resourceProvider, 1, 0)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != "admin-create" || events[1] != "provider-create" {
		t.Fatalf("events = %#v", events)
	}
	if len(admin.createRequests) != 1 ||
		!strings.HasPrefix(
			admin.createRequests[0].IdempotencyKey,
			"fleet-manager/fleet-a/generic/",
		) ||
		!admin.createRequests[0].Ephemeral {
		t.Fatalf("create request = %#v", admin.createRequests)
	}
	if admin.createRequests[0].FleetManagerID != "manager-a" || admin.createRequests[0].Provider != "fake" {
		t.Fatalf("create request ownership = %#v", admin.createRequests[0])
	}
	if len(admin.deletedRunnerIDs) != 0 {
		t.Fatalf("deleted runners = %#v", admin.deletedRunnerIDs)
	}
	if resolver.requestedVersion != "1.2.3" ||
		resolver.requestedOS != "linux" ||
		resolver.requestedArch != "amd64" {
		t.Fatalf(
			"artifact request = %s %s/%s",
			resolver.requestedVersion,
			resolver.requestedOS,
			resolver.requestedArch,
		)
	}
	if resourceProvider.createRequest.RunnerID != "runner-1" ||
		resourceProvider.bootstrap.RegistrationToken != "registration-token" ||
		resourceProvider.bootstrap.Tags["fleet_manager_id"] != "manager-a" ||
		resourceProvider.bootstrap.RunnerAPIURL != "https://superplane.example" {
		t.Fatalf("provider requests = %#v %#v", resourceProvider.createRequest, resourceProvider.bootstrap)
	}
}

func TestReconcileLogsProvisioningActions(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.RunnableTasks = 1
	var output bytes.Buffer
	reconciler, err := New(
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		Config{
			FleetID:         "fleet-a",
			FleetManagerID:  "manager-a",
			OperatingSystem: "linux",
			Architecture:    "amd64",
		},
		slog.New(slog.NewJSONHandler(&output, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}

	messages := loggedMessages(t, output.String())
	for _, expected := range []string{
		"describing fleet",
		"getting fleet capacity",
		"listing provider resources",
		"cleaning up terminated provider resources",
		"listing active runners",
		"reconciling fleet capacity",
		"increasing fleet capacity",
		"creating logical runner",
		"resolving runner artifact",
		"building runner bootstrap",
		"creating provider runner",
		"provisioned runner",
		"fleet reconciliation completed",
	} {
		if !slices.Contains(messages, expected) {
			t.Errorf("missing log message %q in %#v", expected, messages)
		}
	}
}

func TestWarmRunnerRemainsWhenNoTasksAreQueued(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.IdleRunners = 1
	admin.activeRunners = []adminclient.Runner{{
		ID:      "runner-idle",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateIdle,
	}}
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		1,
		0,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admin.createRequests) != 0 || len(admin.deletedRunnerIDs) != 0 {
		t.Fatalf(
			"create requests = %#v, deleted runners = %#v",
			admin.createRequests,
			admin.deletedRunnerIDs,
		)
	}
}

func TestProviderCreationFailureTerminatesLogicalRunner(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.RunnableTasks = 1
	resourceProvider := &fakeProvider{createError: errors.New("provider unavailable")}
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		resourceProvider,
		0,
		0,
	)

	if err := reconciler.Reconcile(context.Background()); err == nil {
		t.Fatal("expected provisioning error")
	}
	if len(admin.deletedRunnerIDs) != 1 || admin.deletedRunnerIDs[0] != "runner-1" {
		t.Fatalf("deleted runners = %#v", admin.deletedRunnerIDs)
	}
}

func TestTerminatedRunnerDeletesTaggedProviderResource(t *testing.T) {
	admin := newFakeAdmin()
	admin.terminatedRunners = []adminclient.Runner{{
		ID:      "runner-terminated",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateTerminated,
	}}
	resourceProvider := &fakeProvider{
		resources: []provider.Resource{{
			ID:       "resource-terminated",
			RunnerID: "runner-terminated",
			FleetID:  "fleet-a",
		}},
	}
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		resourceProvider,
		0,
		0,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(resourceProvider.deleted) != 1 ||
		resourceProvider.deleted[0].ID != "resource-terminated" {
		t.Fatalf("deleted resources = %#v", resourceProvider.deleted)
	}
}

func TestProviderDeletionConfirmation(t *testing.T) {
	for _, test := range []struct {
		name           string
		resourceExists bool
		terminatedAgo  time.Duration
		deleteError    error
		wantConfirmed  bool
	}{
		{name: "deletion accepted but resource still listed", resourceExists: true, terminatedAgo: 2 * time.Minute},
		{name: "failed deletion", resourceExists: true, deleteError: errors.New("provider unavailable")},
		{name: "absent resource after grace", terminatedAgo: 2 * time.Minute, wantConfirmed: true},
		{name: "absent resource during grace", terminatedAgo: time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			admin := newFakeAdmin()
			terminatedAt := time.Now().Add(-test.terminatedAgo)
			admin.terminatedRunners = []adminclient.Runner{{
				ID: "runner-terminated", FleetID: "fleet-a",
				State: adminclient.RunnerStateTerminated, TerminatedAt: &terminatedAt,
				Tags: map[string]string{"fleet_manager_id": "manager-a", "runner_provider": "fake"},
			}}
			resourceProvider := &fakeProvider{deleteError: test.deleteError}
			if test.resourceExists {
				resourceProvider.resources = []provider.Resource{{
					ID: "resource-terminated", RunnerID: "runner-terminated", FleetID: "fleet-a",
				}}
			}
			reconciler := newTestReconciler(t, admin, &fakeArtifactResolver{}, resourceProvider, 0, 0)
			err := reconciler.Reconcile(context.Background())
			if !errors.Is(err, test.deleteError) {
				t.Fatalf("reconcile error = %v, want %v", err, test.deleteError)
			}
			confirmed := slices.Contains(admin.providerDeleted, "runner-terminated")
			if confirmed != test.wantConfirmed {
				t.Fatalf("provider deletion confirmed = %v, want %v", confirmed, test.wantConfirmed)
			}
		})
	}
}

func TestSynchronousDeletionConfirmsOnlyRemovedOwnedResources(t *testing.T) {
	deletionFailure := errors.New("deletion failed")
	for _, test := range []struct {
		name           string
		resourceCount  int
		failedResource string
		foreignOwner   bool
		disabledFleet  bool
		wantConfirmed  bool
	}{
		{name: "observed resource removed", resourceCount: 1, wantConfirmed: true},
		{name: "all observed resources removed", resourceCount: 2, wantConfirmed: true},
		{name: "disabled fleet resource removed", resourceCount: 1, disabledFleet: true, wantConfirmed: true},
		{name: "deletion failed", resourceCount: 1, failedResource: "resource-0"},
		{name: "first observed resource remains", resourceCount: 2, failedResource: "resource-0"},
		{name: "one observed resource remains", resourceCount: 2, failedResource: "resource-1"},
		{name: "resource absent during grace"},
		{name: "runner owned by another manager", resourceCount: 1, foreignOwner: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			admin := newFakeAdmin()
			admin.events = &events
			admin.fleet.Enabled = !test.disabledFleet
			terminatedAt := time.Now().Add(-time.Second)
			owner := "manager-a"
			if test.foreignOwner {
				owner = "manager-b"
			}
			admin.terminatedRunners = []adminclient.Runner{{
				ID: "runner-terminated", FleetID: "fleet-a", State: adminclient.RunnerStateTerminated,
				TerminatedAt: &terminatedAt,
				Tags:         map[string]string{"fleet_manager_id": owner, "runner_provider": "fake"},
			}}
			resourceProvider := &fakeProvider{synchronousDeletion: true, events: &events}
			for i := range test.resourceCount {
				resourceProvider.resources = append(resourceProvider.resources, provider.Resource{
					ID: fmt.Sprintf("resource-%d", i), RunnerID: "runner-terminated", FleetID: "fleet-a",
				})
			}
			var wantError error
			if test.failedResource != "" {
				wantError = deletionFailure
				resourceProvider.deleteErrors = map[string]error{test.failedResource: deletionFailure}
			}
			reconciler := newTestReconciler(t, admin, &fakeArtifactResolver{}, resourceProvider, 0, 0)
			if err := reconciler.Reconcile(t.Context()); !errors.Is(err, wantError) {
				t.Fatalf("reconcile error = %v, want %v", err, wantError)
			}
			if len(resourceProvider.deleted) != test.resourceCount {
				t.Fatalf("deleted resource count = %d, want %d", len(resourceProvider.deleted), test.resourceCount)
			}
			confirmed := slices.Contains(admin.providerDeleted, "runner-terminated")
			if confirmed != test.wantConfirmed {
				t.Fatalf("provider deletion confirmed = %v, want %v", confirmed, test.wantConfirmed)
			}
			if !test.wantConfirmed {
				return
			}
			if len(resourceProvider.resources) != 0 {
				t.Fatalf("provider deletion confirmed while resources remain: %#v", resourceProvider.resources)
			}
			var wantEvents []string
			for i := range test.resourceCount {
				wantEvents = append(wantEvents, fmt.Sprintf("provider-delete/resource-%d", i))
			}
			wantEvents = append(wantEvents, "provider-deleted/runner-terminated")
			if !slices.Equal(events, wantEvents) {
				t.Fatalf("deletion events = %v, want %v", events, wantEvents)
			}
		})
	}
}

func TestProviderDeletionConfirmationBeyondFirstPage(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		for _, pageFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%v/pageFails=%v", enabled, pageFails), func(t *testing.T) {
				admin := newFakeAdmin()
				admin.fleet.Enabled = enabled
				terminatedAt := time.Now().Add(-2 * time.Minute)
				confirmedAt := time.Now().Add(-time.Minute)
				confirmed := make([]map[string]any, listLimit)
				for i := range confirmed {
					confirmed[i] = map[string]any{
						"id": fmt.Sprintf("old-runner-%d", i), "state": adminclient.RunnerStateTerminated,
						"terminatedAt": terminatedAt, "providerDeletedAt": confirmedAt,
					}
				}
				var marked []string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					const fleetPath = "/admin/api/installation/fleets/fleet-a"
					var response any
					switch request.URL.Path {
					case fleetPath:
						response = map[string]any{"fleet": admin.fleet}
					case fleetPath + "/capacity":
						response = admin.capacity
					case fleetPath + "/runners":
						if request.URL.Query().Get("states") != adminclient.RunnerStateTerminated {
							response = map[string]any{"runners": []adminclient.Runner{}}
							break
						}
						if request.URL.Query().Get("limit") != "1000" {
							t.Errorf("page limit = %q", request.URL.Query().Get("limit"))
						}
						switch cursor := request.URL.Query().Get("afterId"); cursor {
						case "":
							response = map[string]any{"runners": confirmed, "hasNextPage": true}
						case "old-runner-999":
							if pageFails {
								http.Error(w, "page unavailable", http.StatusServiceUnavailable)
								return
							}
							response = map[string]any{"runners": []map[string]any{
								{"id": "runner-new", "state": adminclient.RunnerStateTerminated, "terminatedAt": terminatedAt,
									"tags": map[string]string{"fleet_manager_id": "manager-a", "runner_provider": "fake"}},
								{"id": "other-manager", "state": adminclient.RunnerStateTerminated, "terminatedAt": terminatedAt,
									"tags": map[string]string{"fleet_manager_id": "manager-b", "runner_provider": "fake"}},
								{"id": "other-provider", "state": adminclient.RunnerStateTerminated, "terminatedAt": terminatedAt,
									"tags": map[string]string{"fleet_manager_id": "manager-a", "runner_provider": "aws"}},
								{"id": "legacy-runner", "state": adminclient.RunnerStateTerminated, "terminatedAt": terminatedAt,
									"tags": map[string]string{"fleet_manager_id": "manager-a"}},
								{"id": "pending-never-registered", "state": adminclient.RunnerStateTerminated, "terminatedAt": terminatedAt},
							}, "hasNextPage": false}
						default:
							t.Errorf("unexpected cursor = %q", cursor)
							http.Error(w, "unexpected cursor", http.StatusBadRequest)
							return
						}
					default:
						if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, ":provider-deleted") {
							t.Errorf("unexpected request = %s %s", request.Method, request.URL.Path)
							http.NotFound(w, request)
							return
						}
						id := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, fleetPath+"/runners/"), ":provider-deleted")
						var owner adminclient.ProviderDeletionRequest
						if err := json.NewDecoder(request.Body).Decode(&owner); err != nil {
							t.Error(err)
						}
						if owner.FleetManagerID != "manager-a" || owner.Provider != "fake" {
							t.Errorf("confirmation ownership = %#v", owner)
						}
						marked = append(marked, id)
						response = map[string]any{"runner": adminclient.Runner{ID: id}}
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				client, err := adminclient.New(server.URL, "test-token", server.Client())
				if err != nil {
					t.Fatal(err)
				}
				reconciler := newTestReconciler(t, client, &fakeArtifactResolver{}, &fakeProvider{}, 0, 0)
				err = reconciler.Reconcile(t.Context())
				if pageFails {
					if !adminclient.IsStatus(err, http.StatusServiceUnavailable) {
						t.Fatalf("reconcile error = %v, want page failure", err)
					}
					if len(marked) != 0 {
						t.Fatalf("confirmed runners = %v, want none", marked)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(marked, []string{"runner-new"}) {
					t.Fatalf("confirmed runners = %v, want only runner-new", marked)
				}
			})
		}
	}
}

func TestDisabledFleetLogsResourceDeletionReason(t *testing.T) {
	admin := newFakeAdmin()
	admin.fleet.Enabled = false
	admin.activeRunners = []adminclient.Runner{{
		ID:      "runner-idle",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateIdle,
	}}
	resourceProvider := &fakeProvider{
		resources: []provider.Resource{{
			ID:       "resource-idle",
			RunnerID: "runner-idle",
			FleetID:  "fleet-a",
		}},
	}
	var output bytes.Buffer
	reconciler, err := New(
		admin,
		&fakeArtifactResolver{},
		resourceProvider,
		Config{
			FleetID:         "fleet-a",
			FleetManagerID:  "manager-a",
			OperatingSystem: "linux",
			Architecture:    "amd64",
		},
		slog.New(slog.NewJSONHandler(&output, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}

	reasons := loggedStringValues(t, output.String(), "reason")
	if !slices.Contains(reasons, "fleet_disabled") {
		t.Fatalf("deletion reasons = %#v", reasons)
	}
	if slices.Contains(reasons, "capacity_reduction") {
		t.Fatalf("deletion reasons = %#v", reasons)
	}
}

func TestTerminatedRunnerCleanupDoesNotDependOnListLimit(t *testing.T) {
	admin := newFakeAdmin()
	for i := range listLimit {
		admin.terminatedRunners = append(admin.terminatedRunners, adminclient.Runner{
			ID:    fmt.Sprintf("old-runner-%d", i),
			State: adminclient.RunnerStateTerminated,
		})
	}
	admin.terminatedRunners = append(admin.terminatedRunners, adminclient.Runner{
		ID:      "runner-with-resource",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateTerminated,
	})
	resourceProvider := &fakeProvider{
		resources: []provider.Resource{{
			ID:       "resource-terminated",
			RunnerID: "runner-with-resource",
			FleetID:  "fleet-a",
		}},
	}
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		resourceProvider,
		0,
		0,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(resourceProvider.deleted) != 1 ||
		resourceProvider.deleted[0].ID != "resource-terminated" {
		t.Fatalf("deleted resources = %#v", resourceProvider.deleted)
	}
}

func TestMaximumCapacityIncludesBusyRunners(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.RunnableTasks = 2
	admin.capacity.BusyRunners = 2
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		0,
		2,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admin.createRequests) != 0 {
		t.Fatalf("create requests = %#v", admin.createRequests)
	}
}

func TestEqualWarmAndMaximumCapacityKeepsFleetFixedSize(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.BusyRunners = 1
	admin.capacity.IdleRunners = 1
	admin.activeRunners = []adminclient.Runner{{
		ID:      "runner-idle",
		FleetID: "fleet-a",
		State:   adminclient.RunnerStateIdle,
	}}
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		2,
		2,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admin.createRequests) != 0 || len(admin.deletedRunnerIDs) != 0 {
		t.Fatalf(
			"create requests = %#v, deleted runners = %#v",
			admin.createRequests,
			admin.deletedRunnerIDs,
		)
	}
}

func TestMaximumCapacityTerminatesExcessIdleRunners(t *testing.T) {
	admin := newFakeAdmin()
	admin.capacity.IdleRunners = 2
	admin.activeRunners = []adminclient.Runner{
		{ID: "runner-oldest", State: adminclient.RunnerStateIdle},
		{ID: "runner-newest", State: adminclient.RunnerStateIdle},
	}
	admin.activeRunners[0].CreatedAt = admin.activeRunners[1].CreatedAt.Add(-1)
	reconciler := newTestReconciler(
		t,
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		1,
		1,
	)

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admin.deletedRunnerIDs) != 1 ||
		admin.deletedRunnerIDs[0] != "runner-oldest" {
		t.Fatalf("deleted runners = %#v", admin.deletedRunnerIDs)
	}
	if len(admin.deleteStates) != 1 || !slices.Equal(admin.deleteStates[0], []string{adminclient.RunnerStateIdle}) {
		t.Fatalf("termination state preconditions = %#v", admin.deleteStates)
	}
}

func TestScaleDownKeepsProviderResourceUnlessRunnerTerminated(t *testing.T) {
	for _, test := range []struct {
		name     string
		response *adminclient.Runner
		err      error
	}{
		{name: "state precondition conflict", err: &adminclient.HTTPError{StatusCode: http.StatusConflict}},
		{name: "successful cancellation response", response: &adminclient.Runner{ID: "runner-idle", State: adminclient.RunnerStateBusy}},
	} {
		t.Run(test.name, func(t *testing.T) {
			admin := newFakeAdmin()
			admin.capacity.IdleRunners = 1
			admin.activeRunners = []adminclient.Runner{{ID: "runner-idle", State: adminclient.RunnerStateIdle}}
			admin.deleteResponse = test.response
			admin.deleteError = test.err
			resourceProvider := &fakeProvider{resources: []provider.Resource{{
				ID: "resource-idle", RunnerID: "runner-idle", FleetID: "fleet-a",
			}}}
			reconciler := newTestReconciler(t, admin, &fakeArtifactResolver{}, resourceProvider, 0, 1)
			if err := reconciler.Reconcile(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(admin.deletedRunnerIDs) != 1 {
				t.Fatalf("termination requests = %#v", admin.deletedRunnerIDs)
			}
			if len(resourceProvider.deleted) != 0 {
				t.Fatalf("deleted resources for active runner = %#v", resourceProvider.deleted)
			}
		})
	}
}

func TestConfiguredPollTimeoutAppliesAfterInitialCapacityRequest(t *testing.T) {
	admin := newFakeAdmin()
	reconciler, err := New(
		admin,
		&fakeArtifactResolver{},
		&fakeProvider{},
		Config{
			FleetID:         "fleet-a",
			FleetManagerID:  "manager-a",
			OperatingSystem: "linux",
			Architecture:    "amd64",
			PollTimeout:     20 * time.Second,
		},
		slog.Default(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admin.capacityWaits) != 2 ||
		admin.capacityWaits[0] != 0 ||
		admin.capacityWaits[1] != 20 {
		t.Fatalf("capacity waits = %#v", admin.capacityWaits)
	}
}

func TestReconciliationLoopUsesStrategySpecificDelay(t *testing.T) {
	tests := []struct {
		name               string
		describeError      error
		interval           time.Duration
		errorRetryInterval time.Duration
	}{
		{
			name:               "periodic reconciliation",
			interval:           30 * time.Millisecond,
			errorRetryInterval: time.Millisecond,
		},
		{
			name:               "error retry",
			describeError:      errors.New("admin unavailable"),
			interval:           time.Millisecond,
			errorRetryInterval: 30 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			admin := newFakeAdmin()
			admin.describeError = test.describeError
			admin.describeCalls = make(chan time.Time, 2)
			reconciler := newTestReconciler(
				t,
				admin,
				&fakeArtifactResolver{},
				&fakeProvider{},
				0,
				0,
			)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			t.Cleanup(func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("reconciliation loop did not stop")
				}
			})
			runConfig := RunConfig{
				Interval:           test.interval,
				ErrorRetryInterval: test.errorRetryInterval,
			}
			go func() {
				defer close(done)
				runOne(ctx, slog.Default(), runConfig, reconciler)
			}()

			first := waitForDescribeCall(t, admin.describeCalls)
			second := waitForDescribeCall(t, admin.describeCalls)
			if delay := second.Sub(first); delay < 25*time.Millisecond {
				t.Fatalf("delay = %s", delay)
			}
		})
	}
}

func waitForDescribeCall(t *testing.T, calls <-chan time.Time) time.Time {
	t.Helper()
	select {
	case calledAt := <-calls:
		return calledAt
	case <-time.After(time.Second):
		t.Fatal("reconciliation did not start")
		return time.Time{}
	}
}

func loggedMessages(t *testing.T, output string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		var entry struct {
			Message string `json:"msg"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		messages = append(messages, entry.Message)
	}
	return messages
}

func loggedStringValues(t *testing.T, output, key string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var values []string
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		if value, ok := entry[key].(string); ok {
			values = append(values, value)
		}
	}
	return values
}

func newFakeAdmin() *fakeAdmin {
	return &fakeAdmin{
		fleet: adminclient.Fleet{
			ID:      "fleet-a",
			Enabled: true,
			Spec: adminclient.FleetSpec{
				OperatingSystem: "linux",
				Architecture:    "amd64",
			},
			RunnerVersion: "1.2.3",
		},
		capacity: adminclient.Capacity{Generation: "generation-1"},
		createResponse: adminclient.CreateRunnerResponse{
			Runner: adminclient.Runner{
				ID:            "runner-1",
				FleetID:       "fleet-a",
				State:         adminclient.RunnerStatePending,
				RunnerVersion: "1.2.3",
				Ephemeral:     true,
			},
			RegistrationToken: "registration-token",
			RunnerAPIURL:      "https://superplane.example/runner/v1",
		},
	}
}

func newTestReconciler(
	t *testing.T,
	admin AdminClient,
	resolver ArtifactResolver,
	resourceProvider provider.Provider,
	warmCapacity int,
	maxCapacity int,
) *Reconciler {
	t.Helper()
	reconciler, err := New(admin, resolver, resourceProvider, Config{
		FleetID:         "fleet-a",
		FleetManagerID:  "manager-a",
		WarmCapacity:    warmCapacity,
		MaxCapacity:     maxCapacity,
		OperatingSystem: "linux",
		Architecture:    "amd64",
	}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	return reconciler
}
