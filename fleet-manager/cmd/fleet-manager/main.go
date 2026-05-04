package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/fleetmanager"
	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/fleet-manager/internal/webhook"
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
	auth := getenv("AUTH_TOKEN", "")
	handler := fleetmanager.NewRouter(srv, fleetmanager.RouterOptions{AuthToken: auth})

	addr := getenv("LISTEN_ADDR", ":8080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
