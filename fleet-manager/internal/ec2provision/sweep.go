package ec2provision

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	drained, err := l.drainTerminationCandidates(ctx, ids, api.DrainReasonUnhealthy)
	if err != nil {
		return nil, fmt.Errorf("drain unhealthy runners: %w", err)
	}
	if len(drained) == 0 {
		return nil, nil
	}
	_, err = l.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: drained})
	if err != nil {
		return nil, fmt.Errorf("terminate unhealthy: %w", err)
	}
	if l.Log != nil {
		l.Log.Info("ec2 terminated unhealthy runners", slog.Int("count", len(drained)), slog.Any("instance_ids", drained))
	}
	return drained, nil
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
