package broker

import (
	"context"
	"log/slog"
	"time"

	"github.com/superplane/runner/task-broker/internal/dispatch"
)

// dispatchSweepLimit bounds how many stranded tasks a single sweep pass
// reserves, so one slow sweep cannot starve other broker work.
const dispatchSweepLimit = 50

// RunDispatchSweepLoop periodically finds queued tasks on dispatch-driven
// fleets (e.g. Lambda) whose dispatch was never attempted or is stale — the
// invoke failed, the broker restarted mid-dispatch, or another replica died
// after reserving the row — and retries them. It is the only mechanism that
// recovers from a failed dispatch; the inline attempt in createTask is best
// effort.
func RunDispatchSweepLoop(ctx context.Context, log *slog.Logger, srv *Server, interval, staleAfter time.Duration) {
	if srv == nil || srv.Dispatch == nil {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sweepDispatchOnce(ctx, log, srv, staleAfter)
		}
	}
}

func sweepDispatchOnce(ctx context.Context, log *slog.Logger, srv *Server, staleAfter time.Duration) {
	candidates, err := srv.Store.ClaimDispatchCandidates(ctx, dispatch.ProvisionerAWSLambda, staleAfter, dispatchSweepLimit)
	if err != nil {
		if log != nil {
			log.Warn("dispatch sweep: claim candidates", slog.Any("err", err))
		}
		return
	}
	for _, c := range candidates {
		d, needsDispatch := srv.Dispatch.For(c.Provisioner, c.LambdaFunctionName)
		if !needsDispatch {
			continue
		}
		dispatchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := d.Dispatch(dispatchCtx, c.FleetID, c.TaskID)
		cancel()
		if err != nil && log != nil {
			log.Warn("dispatch sweep: dispatch failed, will retry next sweep",
				slog.String("task_id", c.TaskID), slog.String("fleet_id", c.FleetID), slog.Any("err", err))
		}
	}
}
