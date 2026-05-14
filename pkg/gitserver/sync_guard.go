package gitserver

import "sync"

// syncContext tracks whether we're currently inside a git-push sync,
// to prevent the reverse sync from firing and creating infinite loops.
var (
	activeSyncsMu sync.Mutex
	activeSyncs   = make(map[string]bool) // canvasID -> true when syncing from git
)

// MarkSyncActive marks that a canvas is being updated from a git push.
func MarkSyncActive(canvasID string) {
	activeSyncsMu.Lock()
	defer activeSyncsMu.Unlock()
	activeSyncs[canvasID] = true
}

// MarkSyncDone marks that the git push sync is complete.
func MarkSyncDone(canvasID string) {
	activeSyncsMu.Lock()
	defer activeSyncsMu.Unlock()
	delete(activeSyncs, canvasID)
}

// IsSyncActive returns true if the canvas is currently being updated from git.
func IsSyncActive(canvasID string) bool {
	activeSyncsMu.Lock()
	defer activeSyncsMu.Unlock()
	return activeSyncs[canvasID]
}
