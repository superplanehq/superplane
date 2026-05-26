package ec2provision

import (
	"context"
	"errors"
	"testing"

	"github.com/superplane/runner/shared/api"
)

type fakeBrokerClient struct {
	counts api.FleetTaskCountsResponse
	err    error
	calls  int
	gotID  string
}

func (f *fakeBrokerClient) FleetTaskCounts(_ context.Context, fleetID string) (api.FleetTaskCountsResponse, error) {
	f.calls++
	f.gotID = fleetID
	return f.counts, f.err
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
