package broker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/superplane/runner/task-broker/internal/dispatch"
)

const (
	dispatchSweepLimit       = 50
	dispatchSweepConcurrency = 10
)

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
	sem := make(chan struct{}, dispatchSweepConcurrency)
	var wg sync.WaitGroup
	for _, c := range candidates {
		d, needsDispatch := srv.Dispatch.For(c.Provisioner, c.LambdaFunctionName)
		if !needsDispatch {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			dispatchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := d.Dispatch(dispatchCtx, c.FleetID, c.TaskID); err != nil && log != nil {
				log.Warn("dispatch sweep: dispatch failed, will retry next sweep",
					slog.String("task_id", c.TaskID), slog.String("fleet_id", c.FleetID), slog.Any("err", err))
			}
		}()
	}
	wg.Wait()
}
