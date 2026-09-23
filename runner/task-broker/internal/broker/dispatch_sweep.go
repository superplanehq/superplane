package broker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	taskstore "github.com/superplane/runner/task-broker/internal/store"
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
	var candidates []taskstore.DispatchCandidate
	for _, provisioner := range srv.Dispatch.Provisioners() {
		found, err := srv.Store.ClaimDispatchCandidates(ctx, provisioner, staleAfter, dispatchSweepLimit)
		if err != nil {
			if log != nil {
				log.Warn("dispatch sweep: claim candidates", slog.String("provisioner", provisioner), slog.Any("err", err))
			}
			continue
		}
		candidates = append(candidates, found...)
	}
	sem := make(chan struct{}, dispatchSweepConcurrency)
	var wg sync.WaitGroup
	for _, c := range candidates {
		d, needsDispatch := srv.Dispatch.For(c.Provisioner, c.DispatchTarget)
		if !needsDispatch {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil && log != nil {
					log.Warn("dispatch sweep: dispatch panicked, will retry next sweep",
						slog.String("task_id", c.TaskID), slog.String("fleet_id", c.FleetID), slog.Any("recover", r))
				}
			}()
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
