package materialize

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// lockBranchMaterialization serializes materialization of a single canvas branch.
//
// The repository materializer processes repository_branch_updated messages
// concurrently, so several deliveries for the same branch can race: each
// transaction reads no existing draft version and then inserts one, and the
// loser hits the idx_workflow_versions_draft_branch unique constraint. The
// failed delivery is retried indefinitely, republishing branch events on every
// attempt. A transaction-scoped advisory lock keyed on (canvas, branch) makes
// concurrent materializations of the same branch run one at a time, so the
// second sees the committed row and updates it instead of inserting a duplicate.
func lockBranchMaterialization(tx *gorm.DB, canvasID uuid.UUID, branch string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", branchMaterializationLockKey(canvasID, branch)).Error
}

func branchMaterializationLockKey(canvasID uuid.UUID, branch string) int64 {
	h := fnv.New64a()
	h.Write(canvasID[:])
	h.Write([]byte(branch))
	return int64(binary.BigEndian.Uint64(h.Sum(nil))) //nolint:gosec // wraparound is fine; we just need a deterministic key
}
