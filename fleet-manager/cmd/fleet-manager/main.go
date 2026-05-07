package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
	"github.com/superplane/runner/fleet-manager/internal/fleetmanager"
	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/shared/webhook"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := getenv("DATABASE_PATH", "./fleet.db")
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Error("mkdir", slog.Any("err", err))
			os.Exit(1)
		}
	}
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		log.Error("open database", slog.Any("err", err))
		os.Exit(1)
	}
	defer st.Close()

	srv := &fleetmanager.Server{
		Store:   st,
		Webhook: webhook.DefaultSender(),
		Log:     log,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var launcher *ec2provision.Launcher
	ecCfg, ecErr := ec2provision.ConfigFromEnv()
	switch {
	case ecErr == nil:
		var err error
		launcher, err = ec2provision.New(context.Background(), ecCfg, log)
		if err != nil {
			log.Error("ec2 provision init", slog.Any("err", err))
			os.Exit(1)
		}
		srv.EC2Launcher = launcher
		srv.TerminateRunnerAfterTaskEnabled = ecCfg.RunnerTerminateAfterEachTask
		if ecCfg.RunnerTerminateAfterEachTask {
			srv.TerminateRunnerInstance = launcher.TerminateInstance
		}
		reconcileEvery := 60 * time.Second
		if v := getenv("EC2_PROVISION_RECONCILE_INTERVAL_SEC", ""); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 15 {
				reconcileEvery = time.Duration(n) * time.Second
			}
		}
		go ec2provision.RunReconcileLoop(ctx, log, reconcileEvery, launcher)
		log.Info("ec2 hot runner pool enabled",
			slog.Int("hot_instance_count", ecCfg.HotInstanceCount),
			slog.String("reconcile_interval", reconcileEvery.String()))
	case errors.Is(ecErr, ec2provision.ErrDisabled):
		// EC2 pool off
	default:
		log.Error("ec2 provision config", slog.Any("err", ecErr))
		os.Exit(1)
	}
	auth := getenv("AUTH_TOKEN", "")
	diagTok := getenv("FLEET_DIAGNOSTICS_TOKEN", "")
	handler := fleetmanager.NewRouter(srv, fleetmanager.RouterOptions{
		AuthToken:          auth,
		DiagnosticsToken: diagTok,
	})

	addr := getenv("LISTEN_ADDR", ":8080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	reapInterval := 15 * time.Second
	if v := getenv("REAP_INTERVAL_SEC", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			reapInterval = time.Duration(n) * time.Second
		}
	}
	go func() {
		t := time.NewTicker(reapInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := st.ReapExpiredLeases(context.Background())
				if err != nil {
					log.Warn("reap leases", slog.Any("err", err))
					continue
				}
				if n > 0 {
					log.Info("reaped expired task leases", slog.Int64("count", n))
				}
			}
		}
	}()

	go func() {
		log.Info("fleet-manager listening", slog.String("addr", addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", slog.Any("err", err))
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
