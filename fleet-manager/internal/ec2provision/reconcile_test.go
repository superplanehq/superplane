package ec2provision

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/smithy-go"
	"github.com/superplane/runner/shared/api"
)

type fakeBrokerClient struct {
	counts       api.FleetTaskCountsResponse
	err          error
	drain        api.DrainRunnersResponse
	drainErr     error
	calls        int
	drainCalls   int
	gotID        string
	drainRequest api.DrainRunnersRequest
}

func (f *fakeBrokerClient) FleetTaskCounts(_ context.Context, fleetID string) (api.FleetTaskCountsResponse, error) {
	f.calls++
	f.gotID = fleetID
	return f.counts, f.err
}

func (f *fakeBrokerClient) DrainRunners(_ context.Context, req api.DrainRunnersRequest) (api.DrainRunnersResponse, error) {
	f.drainCalls++
	f.drainRequest = req
	return f.drain, f.drainErr
}

func TestDesiredWant_HeadroomOff_UsesHotInstanceCount(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{Claimed: 99}}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 4, Headroom: 0, RunnerFleetID: "f"},
		BrokerClient: fake,
	}
	if got := l.desiredWant(context.Background()); got != 4 {
		t.Fatalf("want 4, got %d", got)
	}
	if fake.calls != 0 {
		t.Fatalf("broker should not be called when Headroom=0, got %d calls", fake.calls)
	}
}

func TestDesiredWant_NilBrokerClient_UsesHotInstanceCount(t *testing.T) {
	l := &Launcher{
		Config: Config{HotInstanceCount: 2, Headroom: 5, RunnerFleetID: "f"},
	}
	if got := l.desiredWant(context.Background()); got != 2 {
		t.Fatalf("want 2, got %d", got)
	}
}

func TestDesiredWant_BrokerOK_QueuedPlusClaimedPlusHeadroom(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{Queued: 10, Claimed: 4}}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 1, Headroom: 3, RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}
	if got := l.desiredWant(context.Background()); got != 17 {
		t.Fatalf("want 17 (10 queued + 4 claimed + 3 headroom), got %d", got)
	}
	if fake.calls != 1 || fake.gotID != "fleet-a" {
		t.Fatalf("broker call: calls=%d id=%q", fake.calls, fake.gotID)
	}
}

func TestDesiredCapacity_BrokerOK_IncludesClaimedRunnerIDs(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{
		Queued:           2,
		Claimed:          1,
		ClaimedRunnerIDs: []string{"i-active"},
	}}
	l := &Launcher{
		Config:       Config{Headroom: 1, RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got := l.desiredCapacity(context.Background())

	if got.want != 4 {
		t.Fatalf("want 4 (2 queued + 1 claimed + 1 headroom), got %d", got.want)
	}
	if len(got.claimedRunnerIDs) != 1 || got.claimedRunnerIDs[0] != "i-active" {
		t.Fatalf("claimed runner ids: %#v", got.claimedRunnerIDs)
	}
}

func TestDesiredCapacity_HeadroomOff_UsesHotCountWithClaimedRunnerIDs(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{
		Queued:           10,
		Claimed:          4,
		ClaimedRunnerIDs: []string{"i-active"},
	}}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 2, Headroom: 0, RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got := l.desiredCapacity(context.Background())

	if got.want != 2 {
		t.Fatalf("want hot count 2, got %d", got.want)
	}
	if !got.claimedRunnerIDsReliable {
		t.Fatal("expected claimed runner ids to be reliable")
	}
	if len(got.claimedRunnerIDs) != 1 || got.claimedRunnerIDs[0] != "i-active" {
		t.Fatalf("claimed runner ids: %#v", got.claimedRunnerIDs)
	}
}

func TestDesiredCapacity_BrokerError_DisablesScaleDownProtection(t *testing.T) {
	fake := &fakeBrokerClient{err: errors.New("boom")}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 2, Headroom: 5, RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got := l.desiredCapacity(context.Background())

	if got.want != 2 {
		t.Fatalf("want fallback hot count 2, got %d", got.want)
	}
	if got.claimedRunnerIDsReliable {
		t.Fatal("claimed runner ids should not be reliable after broker error")
	}
}

func TestDesiredWant_BrokerError_FallsBack(t *testing.T) {
	fake := &fakeBrokerClient{err: errors.New("boom")}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 2, Headroom: 5, RunnerFleetID: "f"},
		BrokerClient: fake,
	}
	if got := l.desiredWant(context.Background()); got != 2 {
		t.Fatalf("want fallback 2, got %d", got)
	}
}

func TestDesiredWant_NoTasks_ScalesDownToHeadroom(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{Queued: 0, Claimed: 0}}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 10, Headroom: 2, RunnerFleetID: "f"},
		BrokerClient: fake,
	}
	if got := l.desiredWant(context.Background()); got != 2 {
		t.Fatalf("want 2 (idle pool drops to headroom), got %d", got)
	}
}

func TestDesiredWant_OnlyQueued_PrewarmsForBurst(t *testing.T) {
	fake := &fakeBrokerClient{counts: api.FleetTaskCountsResponse{Queued: 5, Claimed: 0}}
	l := &Launcher{
		Config:       Config{HotInstanceCount: 1, Headroom: 2, RunnerFleetID: "f"},
		BrokerClient: fake,
	}
	if got := l.desiredWant(context.Background()); got != 7 {
		t.Fatalf("want 7 (5 queued + 0 claimed + 2 headroom), got %d", got)
	}
}

func TestTickAll_TicksEveryLauncherInOrder(t *testing.T) {
	// Multi-pool guarantee: one tick visits every launcher exactly once, preserving
	// the order in the input slice. Spy replaces reconcileLauncher to avoid EC2.
	var seen []string
	spy := func(_ context.Context, _ *slog.Logger, l *Launcher) {
		seen = append(seen, l.Config.RunnerFleetID)
	}
	launchers := []*Launcher{
		{Config: Config{RunnerFleetID: "aws-amd64"}},
		{Config: Config{RunnerFleetID: "aws-arm64"}},
		{Config: Config{RunnerFleetID: "aws-gpu"}},
	}

	tickAll(context.Background(), nil, launchers, spy)

	if len(seen) != 3 {
		t.Fatalf("expected 3 ticks, got %d (%v)", len(seen), seen)
	}
	if seen[0] != "aws-amd64" || seen[1] != "aws-arm64" || seen[2] != "aws-gpu" {
		t.Errorf("tick order = %v, want [aws-amd64 aws-arm64 aws-gpu]", seen)
	}
}

func TestTickAll_EmptyLauncherSliceIsNoOp(t *testing.T) {
	called := 0
	spy := func(_ context.Context, _ *slog.Logger, _ *Launcher) { called++ }
	tickAll(context.Background(), nil, nil, spy)
	tickAll(context.Background(), nil, []*Launcher{}, spy)
	if called != 0 {
		t.Errorf("expected 0 ticks, got %d", called)
	}
}

func TestSelectExcessRunnerIDs_SkipsClaimedRunnersThenUsesOldestFirst(t *testing.T) {
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	instances := []managedInstance{
		{id: "i-old-active", launchUTC: base},
		{id: "i-old-idle", launchUTC: base.Add(1 * time.Minute)},
		{id: "i-new-idle", launchUTC: base.Add(2 * time.Minute)},
	}

	got := selectExcessRunnerIDs(instances, []string{"i-old-active"}, 2)

	if len(got) != 2 || got[0] != "i-old-idle" || got[1] != "i-new-idle" {
		t.Fatalf("selected ids: got %#v want [i-old-idle i-new-idle]", got)
	}
}

func TestSelectExcessRunnerIDs_LeavesCapacityWhenOnlyClaimedRunnersRemain(t *testing.T) {
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	instances := []managedInstance{
		{id: "i-active-1", launchUTC: base},
		{id: "i-active-2", launchUTC: base.Add(1 * time.Minute)},
	}

	got := selectExcessRunnerIDs(instances, []string{"i-active-1", "i-active-2"}, 1)

	if len(got) != 0 {
		t.Fatalf("selected ids: got %#v want none", got)
	}
}

func TestDrainTerminationCandidates_ReturnsOnlyBrokerDrainedRunners(t *testing.T) {
	fake := &fakeBrokerClient{drain: api.DrainRunnersResponse{
		Runners: []api.DrainRunnerStatus{
			{RunnerID: "i-idle", State: api.DrainRunnerStateDrained},
			{RunnerID: "i-busy", State: api.DrainRunnerStateBusy, ActiveTaskID: "task-1"},
		},
	}}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got, err := l.drainTerminationCandidates(context.Background(), []string{"i-idle", "i-busy"}, "test")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0] != "i-idle" {
		t.Fatalf("drained ids: got %#v want [i-idle]", got)
	}
	if fake.drainCalls != 1 {
		t.Fatalf("drain calls = %d, want 1", fake.drainCalls)
	}
	if fake.drainRequest.FleetID != "fleet-a" || len(fake.drainRequest.RunnerIDs) != 2 {
		t.Fatalf("drain request: %#v", fake.drainRequest)
	}
}

func TestDrainTerminationCandidates_FailsClosedWhenBrokerDrainFails(t *testing.T) {
	fake := &fakeBrokerClient{drainErr: errors.New("broker down")}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	if _, err := l.drainTerminationCandidates(context.Background(), []string{"i-idle"}, "test"); err == nil {
		t.Fatal("expected drain error")
	}
}

func TestLaunchAdditional_FallsBackToSingleInstanceBatches(t *testing.T) {
	var batchSizes []int
	launch := func(_ context.Context, count int) ([]string, error) {
		batchSizes = append(batchSizes, count)
		if count > 1 {
			return nil, &smithy.GenericAPIError{Code: "InsufficientInstanceCapacity"}
		}
		return []string{"i-ok"}, nil
	}

	if err := launchAdditional(context.Background(), 3, defaultLaunchBatch, launch); err != nil {
		t.Fatalf("launchAdditional: %v", err)
	}
	want := []int{3, 1, 1, 1}
	if len(batchSizes) != len(want) {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
	}
	for i := range want {
		if batchSizes[i] != want[i] {
			t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
		}
	}
}

func TestLaunchAdditional_UsesConfiguredBatchSize(t *testing.T) {
	var batchSizes []int
	launch := func(_ context.Context, count int) ([]string, error) {
		batchSizes = append(batchSizes, count)
		ids := make([]string, count)
		for i := range ids {
			ids[i] = "i-ok"
		}
		return ids, nil
	}

	if err := launchAdditional(context.Background(), 7, defaultLaunchBatch, launch); err != nil {
		t.Fatalf("launchAdditional: %v", err)
	}
	want := []int{5, 2}
	if len(batchSizes) != len(want) {
		t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
	}
	for i := range want {
		if batchSizes[i] != want[i] {
			t.Fatalf("batch sizes = %v, want %v", batchSizes, want)
		}
	}
}
