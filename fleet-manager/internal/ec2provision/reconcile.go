package ec2provision

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/superplane/runner/shared/api"
)

type instanceSnap struct {
	id        string
	launchUTC time.Time
}

type desiredCapacity struct {
	want                     int
	claimedRunnerIDs         []string
	claimedRunnerIDsReliable bool
}

// RunReconcileLoop periodically reconciles every launcher's pool toward its target.
// Each tick walks `launchers` serially; a per-launcher failure is logged and skipped so
// one pool's broker / EC2 hiccup cannot stall reconciles for the other pools. Each
// launcher's target is dynamic when its Config.Headroom > 0 (want = queued + claimed +
// headroom, pulled from task-broker) and falls back to HotInstanceCount on broker
// failure (see desiredWant).
func RunReconcileLoop(ctx context.Context, log *slog.Logger, interval time.Duration, launchers []*Launcher) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	tickAll(ctx, log, launchers, reconcileLauncher)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickAll(ctx, log, launchers, reconcileLauncher)
		}
	}
}

// tickAll runs `tick` against every launcher serially. Extracted so tests can inject a
// spy in place of reconcileLauncher and assert iteration order/coverage without standing
// up an EC2 client.
func tickAll(ctx context.Context, log *slog.Logger, launchers []*Launcher, tick func(context.Context, *slog.Logger, *Launcher)) {
	for _, l := range launchers {
		tick(ctx, log, l)
	}
}

// reconcileLauncher is the default per-launcher tick: bounded-timeout desired capacity +
// reconcile, logging on failure with the owning fleet id for cross-pool diagnostics.
func reconcileLauncher(ctx context.Context, log *slog.Logger, l *Launcher) {
	start := time.Now()
	defer func() {
		if l.Metrics != nil {
			l.Metrics.ReconcileDuration(ctx, l.Config.RunnerFleetID, time.Since(start))
		}
	}()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	desired := l.desiredCapacity(runCtx)
	if err := l.reconcile(runCtx, desired.want, desired.claimedRunnerIDs, desired.claimedRunnerIDsReliable); err != nil && log != nil {
		log.Warn("ec2 reconcile",
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Int("want", desired.want),
			slog.Any("err", err))
	}
}

func (l *Launcher) desiredWant(ctx context.Context) int {
	if l.Config.Headroom <= 0 || l.BrokerClient == nil {
		return l.Config.HotInstanceCount
	}
	counts, err := l.BrokerClient.FleetTaskCounts(ctx, l.Config.RunnerFleetID)
	if err != nil {
		if l.Log != nil {
			l.Log.Warn("ec2 reconcile: broker task-counts failed, falling back to hot instance count",
				slog.Any("err", err),
				slog.String("fleet_id", l.Config.RunnerFleetID),
				slog.Int("fallback_want", l.Config.HotInstanceCount))
		}
		return l.Config.HotInstanceCount
	}
	return counts.Queued + counts.Claimed + l.Config.Headroom
}

func (l *Launcher) desiredCapacity(ctx context.Context) desiredCapacity {
	if l.BrokerClient == nil {
		return desiredCapacity{want: l.Config.HotInstanceCount}
	}
	counts, err := l.BrokerClient.FleetTaskCounts(ctx, l.Config.RunnerFleetID)
	if err != nil {
		if l.Log != nil {
			l.Log.Warn("ec2 reconcile: broker task-counts failed, falling back to hot instance count without scale-down",
				slog.Any("err", err),
				slog.String("fleet_id", l.Config.RunnerFleetID),
				slog.Int("fallback_want", l.Config.HotInstanceCount))
		}
		return desiredCapacity{want: l.Config.HotInstanceCount}
	}
	want := l.Config.HotInstanceCount
	if l.Config.Headroom > 0 {
		want = counts.Queued + counts.Claimed + l.Config.Headroom
	}
	return desiredCapacity{
		want:                     want,
		claimedRunnerIDs:         counts.ClaimedRunnerIDs,
		claimedRunnerIDsReliable: true,
	}
}

// Reconcile sweeps unhealthy runners, then scales pending+running tagged instances toward want.
func (l *Launcher) Reconcile(ctx context.Context, want int) error {
	return l.reconcile(ctx, want, nil, true)
}

// ReconcileKeepingClaimed sweeps unhealthy runners, then scales pending+running tagged
// instances toward want without terminating EC2 instances that currently own claimed
// broker tasks. EC2 runner_id is the instance id in fleet-managed user-data.
func (l *Launcher) ReconcileKeepingClaimed(ctx context.Context, want int, claimedRunnerIDs []string) error {
	return l.reconcile(ctx, want, claimedRunnerIDs, true)
}

func (l *Launcher) reconcile(ctx context.Context, want int, claimedRunnerIDs []string, scaleDownSafe bool) error {
	if want < 0 {
		return fmt.Errorf("negative hot instance count")
	}
	if _, err := l.SweepUnhealthy(ctx); err != nil {
		return fmt.Errorf("health sweep: %w", err)
	}
	instances, err := l.listManagedInstances(ctx)
	if err != nil {
		return fmt.Errorf("describe instances: %w", err)
	}
	have := len(instances)
	if l.Metrics != nil {
		l.Metrics.SetHotInstances(ctx, l.Config.RunnerFleetID, have)
	}
	l.observeInstanceSpinup(ctx, instances)

	if have == want {
		return nil
	}
	if have < want {
		return l.launchAdditional(ctx, want-have)
	}
	return l.scaleDownExcess(ctx, instances, have, want, claimedRunnerIDs, scaleDownSafe)
}

func (l *Launcher) launchAdditional(ctx context.Context, delta int) error {
	return launchAdditional(ctx, delta, defaultLaunchBatch, l.Launch)
}

// launchAdditional scales up in batches. On InsufficientInstanceCapacity it retries one
// instance at a time so partial AZ capacity can still be used.
func launchAdditional(ctx context.Context, delta, batchSize int, launch func(context.Context, int) ([]string, error)) error {
	remaining := delta
	for remaining > 0 {
		count := remaining
		if count > batchSize {
			count = batchSize
		}
		if _, err := launch(ctx, count); err != nil {
			if isInsufficientInstanceCapacity(err) && count > 1 {
				return launchAdditional(ctx, remaining, 1, launch)
			}
			return fmt.Errorf("launch: %w", err)
		}
		remaining -= count
	}
	return nil
}

func (l *Launcher) scaleDownExcess(ctx context.Context, instances []managedInstance, have, want int, claimedRunnerIDs []string, scaleDownSafe bool) error {
	if !scaleDownSafe {
		l.logUnsafeScaleDownSkipped(have, want)
		return nil
	}

	// Scale down by terminating *oldest* safe instances first. Terminating the newest
	// first tended to kill VMs that had just booted and claimed work.
	remove := have - want
	ids := selectExcessRunnerIDs(instances, claimedRunnerIDs, remove)
	if len(ids) == 0 {
		return nil
	}

	drainedIDs, err := l.drainTerminationCandidates(ctx, ids, "scale_down")
	if err != nil {
		return fmt.Errorf("drain runners: %w", err)
	}
	if len(drainedIDs) == 0 {
		l.logBusyScaleDownSkipped(have, want)
		return nil
	}

	if _, err := l.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: drainedIDs}); err != nil {
		return fmt.Errorf("terminate instances: %w", err)
	}
	if l.Log != nil {
		l.Log.Info("ec2 terminating excess runners", slog.Int("count", len(drainedIDs)), slog.Any("instance_ids", drainedIDs))
	}
	return nil
}

func (l *Launcher) logUnsafeScaleDownSkipped(have, want int) {
	if l.Log == nil {
		return
	}
	l.Log.Warn("ec2 scale-down skipped: claimed runner ids unavailable",
		slog.Int("have", have),
		slog.Int("want", want),
		slog.String("fleet_id", l.Config.RunnerFleetID))
}

func (l *Launcher) logBusyScaleDownSkipped(have, want int) {
	if l.Log == nil {
		return
	}
	l.Log.Info("ec2 scale-down skipped: selected runners are busy",
		slog.Int("have", have),
		slog.Int("want", want),
		slog.String("fleet_id", l.Config.RunnerFleetID))
}

func (l *Launcher) drainTerminationCandidates(ctx context.Context, ids []string, reason string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if l.BrokerClient == nil {
		return ids, nil
	}
	resp, err := l.BrokerClient.DrainRunners(ctx, api.DrainRunnersRequest{
		FleetID:   l.Config.RunnerFleetID,
		RunnerIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	drained := make([]string, 0, len(resp.Runners))
	busy := make([]string, 0)
	busyTaskIDs := make(map[string]string)
	for _, runner := range resp.Runners {
		switch runner.State {
		case api.DrainRunnerStateDrained:
			drained = append(drained, runner.RunnerID)
		case api.DrainRunnerStateBusy:
			busy = append(busy, runner.RunnerID)
			if runner.ActiveTaskID != "" {
				busyTaskIDs[runner.RunnerID] = runner.ActiveTaskID
			}
		}
	}
	if l.Log != nil && len(busy) > 0 {
		l.Log.Info("ec2 termination deferred busy runners",
			slog.String("reason", reason),
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Any("runner_ids", busy),
			slog.Any("active_task_ids", busyTaskIDs))
	}
	if reason != "unhealthy" || len(busy) == 0 {
		return drained, nil
	}

	recovered, err := l.BrokerClient.RecoverLostRunners(ctx, api.RecoverLostRunnersRequest{
		FleetID:   l.Config.RunnerFleetID,
		RunnerIDs: busy,
	})
	if err != nil {
		return nil, err
	}
	recoveredRunnerIDs := terminationReadyLostRunnerIDs(busy, busyTaskIDs, recovered.Tasks)
	if l.Log != nil {
		l.Log.Warn("ec2 unhealthy busy runners ready after recovery",
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Any("runner_ids", recoveredRunnerIDs),
			slog.Any("tasks", recovered.Tasks))
	}
	drained = append(drained, recoveredRunnerIDs...)
	return drained, nil
}

func terminationReadyLostRunnerIDs(busyRunnerIDs []string, activeTaskIDs map[string]string, tasks []api.RunnerTaskRecovery) []string {
	recovered := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		id := strings.TrimSpace(task.RunnerID)
		if id == "" {
			continue
		}
		recovered[id] = struct{}{}
	}

	out := make([]string, 0, len(busyRunnerIDs))
	seen := make(map[string]struct{}, len(busyRunnerIDs))
	for _, runnerID := range busyRunnerIDs {
		runnerID = strings.TrimSpace(runnerID)
		if runnerID == "" {
			continue
		}
		if _, ok := seen[runnerID]; ok {
			continue
		}
		_, hasRecoveredTask := recovered[runnerID]
		hasActiveTaskID := strings.TrimSpace(activeTaskIDs[runnerID]) != ""
		if !hasRecoveredTask && !hasActiveTaskID {
			continue
		}
		seen[runnerID] = struct{}{}
		out = append(out, runnerID)
	}
	return out
}

func selectExcessRunnerIDs(instances []managedInstance, claimedRunnerIDs []string, remove int) []string {
	if remove <= 0 {
		return nil
	}
	claimed := make(map[string]struct{}, len(claimedRunnerIDs))
	for _, id := range claimedRunnerIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			claimed[id] = struct{}{}
		}
	}

	live := make([]instanceSnap, 0, len(instances))
	for _, inst := range instances {
		if _, protected := claimed[inst.id]; protected {
			continue
		}
		live = append(live, instanceSnap{id: inst.id, launchUTC: inst.launchUTC})
	}
	sort.Slice(live, func(i, j int) bool {
		return live[i].launchUTC.Before(live[j].launchUTC)
	})

	ids := make([]string, 0, remove)
	for i := 0; i < remove && i < len(live); i++ {
		ids = append(ids, live[i].id)
	}
	return ids
}

func (l *Launcher) listManagedLive(ctx context.Context) ([]instanceSnap, error) {
	in := &ec2.DescribeInstancesInput{
		Filters: managedDescribeFilters(l.Config.RunnerFleetID, []string{"pending", "running"}),
	}
	pager := ec2.NewDescribeInstancesPaginator(l.Client, in)
	var out []instanceSnap
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
				out = append(out, instanceSnap{id: aws.ToString(inst.InstanceId), launchUTC: *inst.LaunchTime})
			}
		}
	}
	return out, nil
}
