package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/superplane/runner/runner/internal/agent"
)

var log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

func main() {
	cfg := agent.DefaultConfig()
	cfg.BaseURL = getTaskBrokerURL()
	cfg.FleetID = getRunnerFleetID()
	cfg.RunnerID = getRunnerID()
	cfg.Token = getAuthToken()
	cfg.Transport = getTransport()

	if v := os.Getenv("POLL_EMPTY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.PollEmpty = time.Duration(ms) * time.Millisecond
		}
	}

	cfg.ExitAfterEachTask = envTruthy("RUNNER_TERMINATE_AFTER_EACH_TASK")
	if v := strings.TrimSpace(os.Getenv("RUNNER_MAX_EXECUTION_SECONDS")); v != "" {
		sec, err := strconv.Atoi(v)
		switch {
		case err != nil:
			log.Warn("invalid RUNNER_MAX_EXECUTION_SECONDS, ignoring", slog.String("value", v), slog.Any("err", err))
		case sec <= 0:
			log.Warn("invalid RUNNER_MAX_EXECUTION_SECONDS, ignoring", slog.String("value", v), slog.Int("parsed_seconds", sec))
		default:
			cfg.MaxExecutionSeconds = sec
		}
	}

	cfg.Log = log
	cfg.CloudWatchLogGroup = strings.TrimSpace(os.Getenv("RUNNER_CLOUDWATCH_LOG_GROUP"))
	cfg.CloudWatchRegion = strings.TrimSpace(os.Getenv("RUNNER_CLOUDWATCH_REGION"))
	cfg.CloudWatchLogStreamPrefix = strings.TrimSpace(os.Getenv("RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX"))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	healthAddr := strings.TrimSpace(os.Getenv("RUNNER_HEALTH_ADDR"))
	if healthAddr == "" {
		healthAddr = "0.0.0.0:9090"
	}
	go serveHealth(ctx, healthAddr)

	a := &agent.Agent{Config: cfg}

	log.Info("runner starting",
		slog.String("runner_id", cfg.RunnerID),
		slog.String("fleet_id", cfg.FleetID),
		slog.String("task_broker", cfg.BaseURL),
		slog.String("transport", transportLabel(cfg)),
		slog.String("health_addr", healthAddr),
		slog.Bool("cloudwatch_logs", strings.TrimSpace(cfg.CloudWatchLogGroup) != ""))

	if removed, err := agent.SweepDockerOrphans(ctx, cfg.RunnerID); err != nil {
		log.Warn("docker_orphan_sweep failed", slog.Any("err", err))
	} else if removed > 0 {
		log.Info("docker_orphan_sweep", slog.Int("removed", removed))
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		attrs := []any{
			slog.String("main_path", bi.Main.Path),
			slog.String("main_version", bi.Main.Version),
			slog.String("go_version", bi.GoVersion),
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision", "vcs.time", "vcs.modified":
				attrs = append(attrs, slog.String(strings.ReplaceAll(s.Key, ".", "_"), s.Value))
			}
		}
		log.Info("runner build", attrs...)
	}
	if err := a.Run(ctx); err != nil && err != context.Canceled {
		log.Error("runner stopped", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("runner stopped")
}

func serveHealth(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("health server", slog.Any("err", err))
	}
}

func envTruthy(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func transportLabel(cfg agent.Config) string {
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case "http", "polling", "legacy":
		return "http"
	default:
		return "websocket"
	}
}

func getRunnerID() string {
	runnerID := strings.TrimSpace(os.Getenv("RUNNER_ID"))
	if runnerID == "" {
		if h, err := os.Hostname(); err == nil && h != "" {
			runnerID = h
		} else {
			runnerID = "runner-" + uuid.NewString()
		}
	}
	return runnerID
}

func getTaskBrokerURL() string {
	base := strings.TrimSpace(os.Getenv("TASK_BROKER_URL"))
	if base == "" {
		log.Error("TASK_BROKER_URL is required")
		os.Exit(1)
	}
	return base
}

func getRunnerFleetID() string {
	fleetID := strings.TrimSpace(os.Getenv("RUNNER_FLEET_ID"))
	if fleetID == "" {
		log.Error("RUNNER_FLEET_ID is required")
		os.Exit(1)
	}
	return fleetID
}

func getAuthToken() string {
	token := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	if token == "" {
		log.Error("AUTH_TOKEN is required")
		os.Exit(1)
	}
	return token
}

func getTransport() string {
	transport := strings.TrimSpace(os.Getenv("RUNNER_TRANSPORT"))
	if transport == "" {
		return "websocket"
	}
	return transport
}
