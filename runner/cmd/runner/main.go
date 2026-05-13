package main

import (
	"context"
	"log/slog"
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

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	base := strings.TrimSpace(os.Getenv("FLEET_MANAGER_URL"))
	if base == "" {
		log.Error("FLEET_MANAGER_URL is required")
		os.Exit(1)
	}

	runnerID := strings.TrimSpace(os.Getenv("RUNNER_ID"))
	if runnerID == "" {
		if h, err := os.Hostname(); err == nil && h != "" {
			runnerID = h
		} else {
			runnerID = "runner-" + uuid.NewString()
		}
	}

	cfg := agent.DefaultConfig()
	cfg.BaseURL = base
	cfg.RunnerID = runnerID
	cfg.Token = os.Getenv("AUTH_TOKEN")
	cfg.Transport = strings.TrimSpace(os.Getenv("RUNNER_TRANSPORT"))
	if v := os.Getenv("POLL_EMPTY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.PollEmpty = time.Duration(ms) * time.Millisecond
		}
	}
	cfg.ExitAfterEachTask = envTruthy("RUNNER_TERMINATE_AFTER_EACH_TASK")
	cfg.Log = log

	a := &agent.Agent{Config: cfg}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("runner starting", slog.String("runner_id", runnerID), slog.String("fleet_manager", base), slog.String("transport", transportLabel(cfg)))
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
