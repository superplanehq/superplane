package main

import (
	"log"
	"net/http"

	"github.com/superplanehq/superplane/supergit/internal/api"
	"github.com/superplanehq/superplane/supergit/internal/config"
	"github.com/superplanehq/superplane/supergit/internal/storage"
)

func main() {
	cfg := config.Load()

	store, err := storage.NewStore(cfg.Root, cfg.DefaultBranch, storage.Limits{
		MaxFileBytes:   cfg.MaxFileBytes,
		MaxCommitBytes: cfg.MaxCommitBytes,
	})
	if err != nil {
		log.Fatalf("initialize storage: %v", err)
	}

	server := api.NewServer(store, cfg)
	addr := ":" + cfg.Port
	log.Printf("supergit listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
