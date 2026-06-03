package ec2provision

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (l *Launcher) trackPendingLaunches(instanceIDs []string, requestedAt time.Time) {
	if len(instanceIDs) == 0 {
		return
	}
	l.pendingMu.Lock()
	defer l.pendingMu.Unlock()
	if l.pending == nil {
		l.pending = make(map[string]time.Time)
	}
	for _, id := range instanceIDs {
		l.pending[id] = requestedAt
	}
}

// observeInstanceSpinup records instance.spinup.duration{phase=instance_running} when a
// tracked instance transitions to EC2 running, and drops stale pending entries.
func (l *Launcher) observeInstanceSpinup(ctx context.Context, instances []managedInstance) {
	live := make(map[string]string, len(instances))
	for _, inst := range instances {
		live[inst.id] = inst.state
	}

	l.pendingMu.Lock()
	defer l.pendingMu.Unlock()
	if len(l.pending) == 0 {
		return
	}

	fleetID := l.Config.RunnerFleetID
	for id, requestedAt := range l.pending {
		state, ok := live[id]
		if !ok {
			delete(l.pending, id)
			continue
		}
		if state != string(types.InstanceStateNameRunning) {
			continue
		}
		if l.Metrics != nil {
			l.Metrics.InstanceSpinupDuration(ctx, fleetID, "instance_running", time.Since(requestedAt))
		}
		delete(l.pending, id)
	}
}
