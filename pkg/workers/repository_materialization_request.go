package workers

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

// inProcessMaterializer, when set, makes RequestBranchMaterialization run
// materialization synchronously in-process instead of publishing a
// repository_branch_updated message for the worker to consume. Production never
// sets this: handlers stay worker-authoritative and the worker does the work.
// Tests wire it (via support.Setup) so that materialized state is observable
// without running the RabbitMQ worker, while the request path still goes through
// the exact same "register pending row + request materialization" flow.
var (
	inProcessMu           sync.RWMutex
	inProcessMaterializer *materialize.BranchMaterializer
)

// SetInProcessMaterializer installs (or clears, with nil) the synchronous
// in-process materializer. Test-only; never called from production code.
func SetInProcessMaterializer(m *materialize.BranchMaterializer) {
	inProcessMu.Lock()
	defer inProcessMu.Unlock()
	inProcessMaterializer = m
}

// RequestBranchMaterialization asks the materializer worker to project the tip of
// branch into the database. Handlers call this after committing to git and
// registering a pending workflow_versions row, so the expensive snapshot load and
// node/webhook reconciliation happens in the worker instead of on the request
// path. In tests an in-process materializer runs the same work synchronously.
func RequestBranchMaterialization(ctx context.Context, canvasID uuid.UUID, branch, headSHA string, pushedBy *uuid.UUID) error {
	inProcessMu.RLock()
	m := inProcessMaterializer
	inProcessMu.RUnlock()
	if m != nil {
		return m.MaterializeBranch(ctx, canvasID, branch, headSHA, pushedBy)
	}

	pushedByID := ""
	if pushedBy != nil {
		pushedByID = pushedBy.String()
	}

	return messages.NewRepositoryBranchUpdatedMessage(
		canvasID.String(),
		branch,
		headSHA,
		pb.MaterializationStatus_MATERIALIZATION_STATUS_PENDING,
		"",
		pushedByID,
	).PublishBranchUpdated()
}
