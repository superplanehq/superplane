package ec2provision

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/superplane/runner/shared/api"
)

type managedInstance struct {
	id        string
	launchUTC time.Time
	state     string
	privateIP string
}

// SweepUnhealthy terminates running managed instances that fail GET /healthz on the private IP
// after the boot grace period.
func (l *Launcher) SweepUnhealthy(ctx context.Context) ([]string, error) {
	if err := l.retryPendingUnhealthyTerminations(ctx); err != nil && l.Log != nil {
		l.Log.Warn("ec2 unhealthy termination confirmation retry failed",
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Any("err", err))
	}

	instances, err := l.listManagedInstances(ctx)
	if err != nil {
		return nil, err
	}
	grace := time.Duration(l.Config.BootGraceSec) * time.Second
	port := l.Config.RunnerHealthPort
	if port <= 0 {
		port = defaultRunnerHealthPort
	}
	client := &http.Client{Timeout: l.runnerHealthTimeout()}

	var terminate []string
	for _, inst := range instances {
		if inst.state != string(types.InstanceStateNameRunning) {
			continue
		}
		if time.Since(inst.launchUTC) < grace {
			continue
		}
		if inst.privateIP == "" {
			if l.recordHealthFailure(inst.id, "missing_private_ip") {
				terminate = append(terminate, inst.id)
			}
			continue
		}
		url := fmt.Sprintf("http://%s:%d/healthz", inst.privateIP, port)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			if l.recordHealthFailure(inst.id, "build_request") {
				terminate = append(terminate, inst.id)
			}
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			if l.recordHealthFailure(inst.id, err.Error()) {
				terminate = append(terminate, inst.id)
			}
			continue
		}
		ok := resp.StatusCode == http.StatusOK
		_ = resp.Body.Close()
		if !ok {
			if l.recordHealthFailure(inst.id, fmt.Sprintf("status_%d", resp.StatusCode)) {
				terminate = append(terminate, inst.id)
			}
			continue
		}
		l.recordHealthSuccess(inst.id)
	}
	if len(terminate) == 0 {
		return nil, nil
	}
	return l.terminateUnhealthyRunners(ctx, terminate)
}

func (l *Launcher) runnerHealthTimeout() time.Duration {
	seconds := l.Config.RunnerHealthTimeoutSec
	if seconds <= 0 {
		seconds = defaultRunnerHealthTimeoutSec
	}
	return time.Duration(seconds) * time.Second
}

func (l *Launcher) runnerHealthFailureThreshold() int {
	threshold := l.Config.RunnerHealthFailureThreshold
	if threshold <= 0 {
		return defaultRunnerHealthFailureThreshold
	}
	return threshold
}

func (l *Launcher) recordHealthFailure(runnerID, reason string) bool {
	l.healthMu.Lock()
	defer l.healthMu.Unlock()
	if l.healthFailures == nil {
		l.healthFailures = make(map[string]int)
	}
	count := l.healthFailures[runnerID] + 1
	l.healthFailures[runnerID] = count
	threshold := l.runnerHealthFailureThreshold()
	if l.Log != nil {
		l.Log.Warn("runner health probe failed",
			slog.String("runner_id", runnerID),
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.String("reason", reason),
			slog.Int("failure_count", count),
			slog.Int("failure_threshold", threshold))
	}
	return count >= threshold
}

func (l *Launcher) recordHealthSuccess(runnerID string) {
	l.healthMu.Lock()
	defer l.healthMu.Unlock()
	if l.healthFailures == nil {
		return
	}
	count := l.healthFailures[runnerID]
	if count == 0 {
		return
	}
	delete(l.healthFailures, runnerID)
	if l.Log != nil {
		l.Log.Info("runner health probe recovered",
			slog.String("runner_id", runnerID),
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Int("previous_failure_count", count))
	}
}

func (l *Launcher) terminateUnhealthyRunners(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	drained, err := l.drainTerminationCandidates(ctx, ids, api.DrainReasonUnhealthy, false)
	if err != nil {
		return nil, fmt.Errorf("drain unhealthy runners: %w", err)
	}
	if len(drained) == 0 {
		return nil, nil
	}
	if err := l.terminateInstances(ctx, drained); err != nil {
		return nil, fmt.Errorf("terminate unhealthy: %w", err)
	}
	l.rememberPendingUnhealthyTerminations(drained)
	waitCtx, cancel := context.WithTimeout(ctx, terminatedWaitTimeout)
	defer cancel()
	if err := l.waitForInstancesTerminated(waitCtx, drained); err != nil {
		return nil, fmt.Errorf("wait for unhealthy termination: %w", err)
	}
	if err := l.confirmTerminatedUnhealthyRunners(ctx, drained); err != nil {
		return nil, fmt.Errorf("confirm unhealthy termination: %w", err)
	}
	if l.Log != nil {
		l.Log.Info("ec2 terminated unhealthy runners", slog.Int("count", len(drained)), slog.Any("instance_ids", drained))
	}
	return drained, nil
}

func (l *Launcher) retryPendingUnhealthyTerminations(ctx context.Context) error {
	ids := l.pendingUnhealthyTerminations()
	if l.BrokerClient != nil {
		counts, err := l.BrokerClient.FleetTaskCounts(ctx, l.Config.RunnerFleetID)
		if err != nil {
			return fmt.Errorf("fleet task counts: %w", err)
		}
		ids = mergeRunnerIDs(ids, counts.ClaimedRunnerIDs)
	}
	if len(ids) == 0 {
		return nil
	}
	return l.confirmTerminatedUnhealthyRunners(ctx, ids)
}

func (l *Launcher) rememberPendingUnhealthyTerminations(ids []string) {
	if len(ids) == 0 {
		return
	}
	l.unhealthyTerminationMu.Lock()
	defer l.unhealthyTerminationMu.Unlock()
	if l.unhealthyTerminationPending == nil {
		l.unhealthyTerminationPending = make(map[string]struct{}, len(ids))
	}
	for _, id := range ids {
		l.unhealthyTerminationPending[id] = struct{}{}
	}
}

func (l *Launcher) forgetPendingUnhealthyTerminations(ids []string) {
	if len(ids) == 0 {
		return
	}
	l.unhealthyTerminationMu.Lock()
	defer l.unhealthyTerminationMu.Unlock()
	for _, id := range ids {
		delete(l.unhealthyTerminationPending, id)
	}
}

func (l *Launcher) pendingUnhealthyTerminations() []string {
	l.unhealthyTerminationMu.Lock()
	defer l.unhealthyTerminationMu.Unlock()
	ids := make([]string, 0, len(l.unhealthyTerminationPending))
	for id := range l.unhealthyTerminationPending {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (l *Launcher) confirmTerminatedUnhealthyRunners(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	terminatedIDs, err := l.terminatedInstanceIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(terminatedIDs) == 0 {
		return nil
	}
	if _, err := l.drainTerminationCandidates(ctx, terminatedIDs, api.DrainReasonUnhealthy, true); err != nil {
		return err
	}
	l.forgetPendingUnhealthyTerminations(terminatedIDs)
	return nil
}

func (l *Launcher) terminateInstances(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	in := &ec2.TerminateInstancesInput{InstanceIds: ids}
	if l.terminateInstancesHook != nil {
		_, err := l.terminateInstancesHook(ctx, in)
		return err
	}
	_, err := l.Client.TerminateInstances(ctx, in)
	return err
}

func (l *Launcher) waitForInstancesTerminated(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	for {
		terminated, err := l.allInstancesTerminated(ctx, ids)
		if err != nil {
			return err
		}
		if terminated {
			return nil
		}
		timer := time.NewTimer(terminatedPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (l *Launcher) allInstancesTerminated(ctx context.Context, ids []string) (bool, error) {
	terminatedIDs, err := l.terminatedInstanceIDs(ctx, ids)
	if err != nil {
		return false, err
	}
	return len(terminatedIDs) == len(ids), nil
}

func (l *Launcher) terminatedInstanceIDs(ctx context.Context, ids []string) ([]string, error) {
	ids = mergeRunnerIDs(ids, nil)
	terminated := make([]string, 0, len(ids))
	for _, id := range ids {
		ok, err := l.instanceTerminated(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			terminated = append(terminated, id)
		}
	}
	return terminated, nil
}

func (l *Launcher) instanceTerminated(ctx context.Context, id string) (bool, error) {
	in := &ec2.DescribeInstancesInput{InstanceIds: []string{id}}
	var out *ec2.DescribeInstancesOutput
	var err error
	if l.describeInstancesHook != nil {
		out, err = l.describeInstancesHook(ctx, in)
	} else {
		out, err = l.Client.DescribeInstances(ctx, in)
	}
	if err != nil {
		if isInstanceNotFound(err) {
			return true, nil
		}
		return false, err
	}

	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			if aws.ToString(instance.InstanceId) == id {
				return instance.State.Name == types.InstanceStateNameTerminated, nil
			}
		}
	}
	return false, nil
}

func isInstanceNotFound(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.ErrorCode() == "InvalidInstanceID.NotFound"
}

func mergeRunnerIDs(first, second []string) []string {
	out := make([]string, 0, len(first)+len(second))
	seen := make(map[string]struct{}, len(first)+len(second))
	for _, id := range append(first, second...) {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (l *Launcher) listManagedInstances(ctx context.Context) ([]managedInstance, error) {
	in := &ec2.DescribeInstancesInput{
		Filters: managedDescribeFilters(l.Config.RunnerFleetID, []string{"pending", "running"}),
	}
	pager := ec2.NewDescribeInstancesPaginator(l.Client, in)
	var out []managedInstance
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rv := range page.Reservations {
			for _, inst := range rv.Instances {
				if inst.InstanceId == nil || inst.LaunchTime == nil {
					continue
				}
				row := managedInstance{
					id:        aws.ToString(inst.InstanceId),
					launchUTC: *inst.LaunchTime,
					state:     string(inst.State.Name),
				}
				if inst.PrivateIpAddress != nil {
					row.privateIP = aws.ToString(inst.PrivateIpAddress)
				}
				out = append(out, row)
			}
		}
	}
	return out, nil
}
