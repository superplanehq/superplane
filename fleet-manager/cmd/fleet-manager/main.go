package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/brokerclient"
	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
	"github.com/superplane/runner/fleet-manager/internal/fleetmanager"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	srv := &fleetmanager.Server{Log: log}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ecCfg, ecErr := ec2provision.ConfigFromEnv()
	switch {
	case ecErr == nil:
		launcher, err := ec2provision.New(context.Background(), ecCfg, log)
		if err != nil {
			log.Error("ec2 provision init", slog.Any("err", err))
			os.Exit(1)
		}
		srv.EC2Launcher = launcher
		if ecCfg.Headroom > 0 {
			launcher.BrokerClient = brokerclient.New(ecCfg.TaskBrokerURL, ecCfg.RunnersAuthToken)
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
			slog.Int("runner_headroom", ecCfg.Headroom),
			slog.Bool("dynamic_scaling", ecCfg.Headroom > 0),
			slog.String("reconcile_interval", reconcileEvery.String()))
	case errors.Is(ecErr, ec2provision.ErrDisabled):
		log.Info("ec2 provisioning disabled — fleet-manager serves diagnostics only when EC2_PROVISION_* is configured")
	default:
		log.Error("ec2 provision config", slog.Any("err", ecErr))
		os.Exit(1)
	}

	auth := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	diagTok := strings.TrimSpace(os.Getenv("FLEET_DIAGNOSTICS_TOKEN"))
	handler := fleetmanager.NewRouter(srv, fleetmanager.RouterOptions{
		AuthToken:        auth,
		DiagnosticsToken: diagTok,
	})

	addr := getenv("LISTEN_ADDR", ":8080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

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
