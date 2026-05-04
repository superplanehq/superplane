package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
	if v := os.Getenv("POLL_EMPTY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.PollEmpty = time.Duration(ms) * time.Millisecond
		}
	}

	a := &agent.Agent{Config: cfg}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("runner starting", slog.String("runner_id", runnerID), slog.String("fleet_manager", base))
	if err := a.Run(ctx); err != nil && err != context.Canceled {
		log.Error("runner stopped", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("runner stopped")
}
