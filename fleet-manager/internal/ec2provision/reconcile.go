package ec2provision

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type instanceSnap struct {
	id        string
	launchUTC time.Time
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

// reconcileLauncher is the default per-launcher tick: bounded-timeout desiredWant +
// Reconcile, logging on failure with the owning fleet id for cross-pool diagnostics.
func reconcileLauncher(ctx context.Context, log *slog.Logger, l *Launcher) {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	want := l.desiredWant(runCtx)
	if err := l.Reconcile(runCtx, want); err != nil && log != nil {
		log.Warn("ec2 reconcile",
			slog.String("fleet_id", l.Config.RunnerFleetID),
			slog.Int("want", want),
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

// Reconcile sweeps unhealthy runners, then scales pending+running tagged instances toward want.
func (l *Launcher) Reconcile(ctx context.Context, want int) error {
	if want < 0 {
		return fmt.Errorf("negative hot instance count")
	}
	if _, err := l.SweepUnhealthy(ctx); err != nil {
		return fmt.Errorf("health sweep: %w", err)
	}
	live, err := l.listManagedLive(ctx)
	if err != nil {
		return fmt.Errorf("describe instances: %w", err)
	}
	have := len(live)

	switch {
	case have < want:
		delta := want - have
		for delta > 0 {
			n := delta
			if n > maxLaunch {
				n = maxLaunch
			}
			_, err := l.Launch(ctx, n)
			if err != nil {
				return fmt.Errorf("launch: %w", err)
			}
			delta -= n
		}
	case have > want:
		// Scale down by terminating *oldest* instances first. Terminating the newest first
		// tended to kill VMs that had just booted and claimed work → PTY/read EIO and flaky tasks.
		sort.Slice(live, func(i, j int) bool {
			return live[i].launchUTC.Before(live[j].launchUTC)
		})
		remove := have - want
		ids := make([]string, 0, remove)
		for i := 0; i < remove && i < len(live); i++ {
			ids = append(ids, live[i].id)
		}
		if len(ids) == 0 {
			return nil
		}
		_, err := l.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: ids})
		if err != nil {
			return fmt.Errorf("terminate instances: %w", err)
		}
		if l.Log != nil {
			l.Log.Info("ec2 terminating excess runners", slog.Int("count", len(ids)), slog.Any("instance_ids", ids))
		}
	}
	return nil
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
