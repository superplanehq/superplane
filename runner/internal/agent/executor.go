package agent

import (
	"context"
	"io"

	"github.com/superplane/runner/shared/api"
)

// Executor runs one task end-to-end and is responsible for its own setup
// and teardown. Concrete implementations exist for host execution
// (HostExecutor) and containerized execution (DockerExecutor).
//
// Implementations must ensure that any resources they create (e.g. Docker
// containers) are released even when ctx is cancelled or Execute returns
// an error — typically by performing cleanup inside a deferred function.
//
// live, when non-nil, receives a copy of every stdout/stderr byte the task
// produces as it runs, in addition to whatever the implementation accumulates
// for the returned `output` string (used by the agent for CloudWatch live
// log streaming).
type Executor interface {
	Execute(ctx context.Context, task *api.TaskPayload, live io.Writer) (exit int, output string, err error)
}
