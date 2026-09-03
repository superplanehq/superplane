package workers

import (
	"context"

	"golang.org/x/sync/semaphore"
)

// Workers bound their in-flight tasks with a weighted semaphore. The limits are
// named here so that drainTasks always waits for the same number of slots the
// worker is able to hand out. A literal in one place and a different literal in
// the other would drain part of the work and report success.
const (
	maxConcurrentTasks        = 25
	maxConcurrentCleanupTasks = 10
)

// drainTasks waits for the tasks a worker has already started. Acquiring the
// whole weight succeeds only once every in-flight task has released its slot,
// so a cancelled worker finishes its work instead of abandoning it in the
// middle of a database transaction or an external API call.
//
// The caller bounds this: the shutdown deadline in pkg/server gives up on the
// drain and exits, so a task that never returns cannot hold the process open.
func drainTasks(sem *semaphore.Weighted, limit int64) {
	//
	// context.Background, not the worker's context: that context is already
	// cancelled by the time we drain, and passing it would make Acquire return
	// immediately without waiting for anything.
	//
	_ = sem.Acquire(context.Background(), limit)
}
