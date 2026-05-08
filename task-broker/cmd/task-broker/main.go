package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/superplane/runner/task-broker/internal/broker"

	"github.com/superplane/runner/shared/webhook"
	"github.com/superplane/runner/task-broker/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := getenv("DATABASE_PATH", "./broker.db")
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

	publicURL := getenv("BROKER_PUBLIC_URL", "")
	if publicURL == "" {
		log.Warn("BROKER_PUBLIC_URL unset; downstream fleet-man cannot POST completion webhooks in production")
	}

	ws := webhook.DefaultSender()
	ws.Log = log
	srv := &broker.Server{
		Store:     st,
		PublicURL: publicURL,
		Webhook:   ws,
		Log:       log,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	auth := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	if auth == "" {
		log.Error("AUTH_TOKEN is required — set a non-empty secret; clients send Authorization: Bearer <token> for /v1 (except webhook callbacks)")
		os.Exit(1)
	}
	handler := broker.NewRouter(srv, broker.RouterOptions{AuthToken: auth})

	addr := getenv("LISTEN_ADDR", ":8081")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("task-broker listening", slog.String("addr", addr))
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
