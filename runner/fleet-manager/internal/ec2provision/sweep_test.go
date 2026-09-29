package ec2provision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/superplane/runner/shared/api"
)

func TestTerminateUnhealthyRunnersDefersInProgressClaim(t *testing.T) {
	fake := &fakeBrokerClient{drain: api.DrainRunnersResponse{
		Runners: []api.DrainRunnerStatus{
			{RunnerID: "i-busy", State: api.DrainRunnerStateBusy},
		},
	}}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got, err := l.terminateUnhealthyRunners(context.Background(), []string{"i-busy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("terminated ids: got %#v want none", got)
	}
	if fake.drainCalls != 1 {
		t.Fatalf("drain calls = %d, want 1", fake.drainCalls)
	}
	if fake.drainRequest.FleetID != "fleet-a" || len(fake.drainRequest.RunnerIDs) != 1 {
		t.Fatalf("drain request: %#v", fake.drainRequest)
	}
}

func TestTerminateUnhealthyRunnersConfirmsAfterEC2Terminated(t *testing.T) {
	fake := &fakeBrokerClient{
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
	}
	var terminated []string
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		terminateInstancesHook: func(_ context.Context, in *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
			terminated = append(terminated, in.InstanceIds...)
			return &ec2.TerminateInstancesOutput{}, nil
		},
		describeInstancesHook: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return describeInstancesWithState(in.InstanceIds, types.InstanceStateNameTerminated), nil
		},
	}

	got, err := l.terminateUnhealthyRunners(context.Background(), []string{"i-lost"})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0] != "i-lost" {
		t.Fatalf("terminated ids: %#v", got)
	}
	if len(terminated) != 1 || terminated[0] != "i-lost" {
		t.Fatalf("ec2 terminated ids: %#v", terminated)
	}
	if len(fake.drainRequests) != 2 {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}
	if fake.drainRequests[0].Reason != api.DrainReasonUnhealthy || fake.drainRequests[0].TerminationConfirmed {
		t.Fatalf("pre-confirm drain request: %#v", fake.drainRequests[0])
	}
	if fake.drainRequests[1].Reason != api.DrainReasonUnhealthy || !fake.drainRequests[1].TerminationConfirmed {
		t.Fatalf("confirmation drain request: %#v", fake.drainRequests[1])
	}
}

func TestTerminateUnhealthyRunnersDoesNotConfirmAfterTerminateError(t *testing.T) {
	fake := &fakeBrokerClient{
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
	}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		terminateInstancesHook: func(_ context.Context, _ *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
			return nil, errors.New("ec2 down")
		},
	}

	if _, err := l.terminateUnhealthyRunners(context.Background(), []string{"i-lost"}); err == nil {
		t.Fatal("expected terminate error")
	}
	if len(fake.drainRequests) != 1 {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}
}

func TestTerminateUnhealthyRunnersDoesNotConfirmAfterWaitTimeout(t *testing.T) {
	fake := &fakeBrokerClient{
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
	}
	state := types.InstanceStateNameRunning
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		terminateInstancesHook: func(_ context.Context, _ *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
			return &ec2.TerminateInstancesOutput{}, nil
		},
		describeInstancesHook: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return describeInstancesWithState(in.InstanceIds, state), nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, err := l.terminateUnhealthyRunners(ctx, []string{"i-lost"}); err == nil {
		t.Fatal("expected wait timeout")
	}
	if len(fake.drainRequests) != 1 {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}

	state = types.InstanceStateNameTerminated
	if err := l.retryPendingUnhealthyTerminations(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fake.drainRequests) != 2 || !fake.drainRequests[1].TerminationConfirmed {
		t.Fatalf("drain requests after retry: %#v", fake.drainRequests)
	}
}

func TestTerminateUnhealthyRunnersSurfacesBrokerConfirmationError(t *testing.T) {
	fake := &fakeBrokerClient{
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
		drainErrs: []error{nil, errors.New("broker confirm down")},
	}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		terminateInstancesHook: func(_ context.Context, _ *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
			return &ec2.TerminateInstancesOutput{}, nil
		},
		describeInstancesHook: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return describeInstancesWithState(in.InstanceIds, types.InstanceStateNameTerminated), nil
		},
	}

	if _, err := l.terminateUnhealthyRunners(context.Background(), []string{"i-lost"}); err == nil {
		t.Fatal("expected confirmation error")
	}
	if len(fake.drainRequests) != 2 || !fake.drainRequests[1].TerminationConfirmed {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}
}

func TestRetryPendingUnhealthyTerminationsUsesBrokerClaimedRunnerIDsAfterRestart(t *testing.T) {
	fake := &fakeBrokerClient{
		counts: api.FleetTaskCountsResponse{
			ClaimedRunnerIDs: []string{"i-lost"},
		},
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
	}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		describeInstancesHook: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return describeInstancesWithState(in.InstanceIds, types.InstanceStateNameTerminated), nil
		},
	}

	if err := l.retryPendingUnhealthyTerminations(context.Background()); err != nil {
		t.Fatal(err)
	}

	if fake.calls != 1 || fake.gotID != "fleet-a" {
		t.Fatalf("fleet counts call: calls=%d id=%q", fake.calls, fake.gotID)
	}
	if len(fake.drainRequests) != 1 || !fake.drainRequests[0].TerminationConfirmed {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}
	if len(fake.drainRequests[0].RunnerIDs) != 1 || fake.drainRequests[0].RunnerIDs[0] != "i-lost" {
		t.Fatalf("confirmed runner ids: %#v", fake.drainRequests[0].RunnerIDs)
	}
}

func TestRetryPendingUnhealthyTerminationsTreatsMissingInstanceAsTerminated(t *testing.T) {
	fake := &fakeBrokerClient{
		counts: api.FleetTaskCountsResponse{
			ClaimedRunnerIDs: []string{"i-lost"},
		},
		drainResponses: []api.DrainRunnersResponse{
			{Runners: []api.DrainRunnerStatus{{RunnerID: "i-lost", State: api.DrainRunnerStateDrained}}},
		},
	}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
		describeInstancesHook: func(_ context.Context, _ *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			return nil, &smithy.GenericAPIError{Code: "InvalidInstanceID.NotFound", Message: "gone"}
		},
	}

	if err := l.retryPendingUnhealthyTerminations(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(fake.drainRequests) != 1 || !fake.drainRequests[0].TerminationConfirmed {
		t.Fatalf("drain requests: %#v", fake.drainRequests)
	}
}

func TestRecordHealthFailureRequiresConsecutiveFailures(t *testing.T) {
	l := &Launcher{Config: Config{
		RunnerFleetID:                "fleet-a",
		RunnerHealthFailureThreshold: 3,
	}}

	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("first failure should not terminate")
	}
	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("second failure should not terminate")
	}
	if !l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("third failure should terminate")
	}
}

func TestRecordHealthSuccessResetsConsecutiveFailures(t *testing.T) {
	l := &Launcher{Config: Config{
		RunnerFleetID:                "fleet-a",
		RunnerHealthFailureThreshold: 2,
	}}

	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("first failure should not terminate")
	}
	l.recordHealthSuccess("i-runner")
	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("failure after recovery should restart the count")
	}
}

func describeInstancesWithState(ids []string, state types.InstanceStateName) *ec2.DescribeInstancesOutput {
	instances := make([]types.Instance, 0, len(ids))
	for _, id := range ids {
		instances = append(instances, types.Instance{
			InstanceId: aws.String(id),
			State:      &types.InstanceState{Name: state},
		})
	}
	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{Instances: instances}},
	}
}
