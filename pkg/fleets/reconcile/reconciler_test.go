package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

type fakeAdmin struct {
	fleet             adminclient.Fleet
	capacity          adminclient.Capacity
	activeRunners     []adminclient.Runner
	terminatedRunners []adminclient.Runner
	createResponse    adminclient.CreateRunnerResponse
	createRequests    []adminclient.CreateRunnerRequest
	deletedRunnerIDs  []string
	events            *[]string
}

func (f *fakeAdmin) DescribeFleet(context.Context, string) (adminclient.Fleet, error) {
	return f.fleet, nil
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
	context.Context,
	string,
	string,
	int,
) (adminclient.Capacity, error) {
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

func (f *fakeAdmin) DeleteRunner(
	_ context.Context,
	_, runnerID string,
) (adminclient.Runner, error) {
	f.deletedRunnerIDs = append(f.deletedRunnerIDs, runnerID)
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
	resources     []provider.Resource
	createRequest provider.CreateRequest
	bootstrap     provider.RunnerBootstrap
	createError   error
	deleted       []provider.Resource
	events        *[]string
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
	return nil
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
		resourceProvider.bootstrap.RunnerAPIURL != "https://superplane.example" {
		t.Fatalf("provider requests = %#v %#v", resourceProvider.createRequest, resourceProvider.bootstrap)
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
		FleetID:             "fleet-a",
		WarmCapacity:        warmCapacity,
		MaxCapacity:         maxCapacity,
		OperatingSystem:     "linux",
		Architecture:        "amd64",
		CapacityWaitSeconds: 0,
	}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	return reconciler
}
