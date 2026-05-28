package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/brokerclient"
	"github.com/superplane/runner/fleet-manager/internal/config"
	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
	"github.com/superplane/runner/fleet-manager/internal/fleetmanager"
)

// FM_CONFIG_FILE is the only environment variable the binary reads. Everything else
// (broker URL, tokens, listen addr, per-pool AMIs, etc.) lives in the JSON config file.
// Default path matches the bind-mount in the Docker deploy script.
const (
	envConfigFile     = "FM_CONFIG_FILE"
	defaultConfigPath = "/etc/fleet-manager/config.json"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfgPath := os.Getenv(envConfigFile)
	if cfgPath == "" {
		cfgPath = defaultConfigPath
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Error("config load", slog.String("path", cfgPath), slog.Any("err", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &fleetmanager.Server{Log: log}

	// Build one Launcher per pool. Headroom-enabled pools share a single broker client
	// (same broker URL + token across all pools in this FM process).
	var sharedBroker *brokerclient.Client
	if hasDynamicPool(cfg) {
		sharedBroker = brokerclient.New(cfg.TaskBrokerURL, cfg.RunnerAuthToken)
	}
	launchers := make([]*ec2provision.Launcher, 0, len(cfg.Pools))
	for _, p := range cfg.Pools {
		poolCfg := cfg.ToPoolConfig(p)
		l, err := ec2provision.New(context.Background(), poolCfg, log)
		if err != nil {
			log.Error("ec2 launcher init",
				slog.String("fleet_id", p.FleetID),
				slog.Any("err", err))
			os.Exit(1)
		}
		if p.Headroom > 0 {
			l.BrokerClient = sharedBroker
		}
		launchers = append(launchers, l)
	}
	srv.EC2Launchers = launchers

	reconcileEvery := time.Duration(cfg.ReconcileIntervalSec) * time.Second
	go ec2provision.RunReconcileLoop(ctx, log, reconcileEvery, launchers)
	log.Info("fleet-manager started",
		slog.String("config_path", cfgPath),
		slog.Int("pools", len(launchers)),
		slog.String("aws_region", cfg.AWSRegion),
		slog.String("reconcile_interval", reconcileEvery.String()))
	for _, p := range cfg.Pools {
		log.Info("pool configured",
			slog.String("fleet_id", p.FleetID),
			slog.String("instance_type", p.InstanceType),
			slog.Int("hot_instance_count", p.HotInstanceCount),
			slog.Int("headroom", p.Headroom),
			slog.Bool("dynamic_scaling", p.Headroom > 0))
	}

	handler := fleetmanager.NewRouter(srv, fleetmanager.RouterOptions{
		AuthToken:        cfg.AuthToken,
		DiagnosticsToken: cfg.DiagnosticsToken,
	})
	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("fleet-manager listening", slog.String("addr", cfg.ListenAddr))
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

// hasDynamicPool reports whether any pool wants dynamic scaling (Headroom > 0).
// Used to decide whether to instantiate the shared broker client at startup.
func hasDynamicPool(cfg *config.File) bool {
	for _, p := range cfg.Pools {
		if p.Headroom > 0 {
			return true
		}
	}
	return false
}
