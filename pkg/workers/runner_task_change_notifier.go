package workers

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// RunnerTaskChangeNotifier collects queue and cancellation changes inside one
// transaction. The transaction owner publishes only after a successful commit.
type RunnerTaskChangeNotifier struct {
	organizations map[uuid.UUID]struct{}
	runners       map[uuid.UUID]struct{}
}

func (n *RunnerTaskChangeNotifier) Bind(ctx *contexts.RunnerTaskContext) *contexts.RunnerTaskContext {
	if n == nil {
		return ctx
	}
	return ctx.WithOnChanged(n.Record)
}

func (n *RunnerTaskChangeNotifier) Record(organizationID uuid.UUID, runnerID *uuid.UUID) {
	if n == nil {
		return
	}
	if n.organizations == nil {
		n.organizations = map[uuid.UUID]struct{}{}
	}
	n.organizations[organizationID] = struct{}{}
	if runnerID == nil {
		return
	}
	if n.runners == nil {
		n.runners = map[uuid.UUID]struct{}{}
	}
	n.runners[*runnerID] = struct{}{}
}

func (n *RunnerTaskChangeNotifier) Publish() {
	if n == nil {
		return
	}
	for organizationID := range n.organizations {
		if err := runnercontrol.PublishOrganizationCapacity(context.Background(), organizationID); err != nil {
			log.WithError(err).WithField("organization", organizationID).Warn("Failed to publish runner queue change")
		}
	}
	for runnerID := range n.runners {
		if err := runnercontrol.Publish(runnercontrol.Notification{RunnerID: runnerID.String()}); err != nil {
			log.WithError(err).WithField("runner", runnerID).Warn("Failed to publish runner cancellation")
		}
	}
}
