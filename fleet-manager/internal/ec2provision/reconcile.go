package ec2provision

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type instanceSnap struct {
	id        string
	launchUTC time.Time
}

// RunReconcileLoop periodically matches managed instances to Config.HotInstanceCount.
func RunReconcileLoop(ctx context.Context, log *slog.Logger, interval time.Duration, l *Launcher) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	tick := func() {
		runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := l.Reconcile(runCtx, l.Config.HotInstanceCount)
		cancel()
		if err != nil && log != nil {
			log.Warn("ec2 reconcile", slog.Any("err", err), slog.Int("want", l.Config.HotInstanceCount))
		}
	}
	tick()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
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
		Filters: []types.Filter{
			{Name: aws.String("tag:" + TagKeyManaged), Values: []string{"true"}},
			{Name: aws.String("instance-state-name"), Values: []string{"pending", "running"}},
		},
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
