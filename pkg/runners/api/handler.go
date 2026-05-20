package api

import (
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/runners"
)

// MaxRequestBodyBytes limits runner webhook and fleet-bridge request bodies.
const MaxRequestBodyBytes = 64 * 1024

// Handler serves runner fleet admin, bridge, and live-log HTTP endpoints.
type Handler struct {
	BaseURL     string
	Registry    *registry.Registry
	AuthService authorization.Authorization
	Store       runners.Store
}

// Config wires dependencies for a Handler.
type Config struct {
	BaseURL     string
	Registry    *registry.Registry
	AuthService authorization.Authorization
	Store       runners.Store
}

// New returns a Handler with a Postgres runner store when Store is nil.
func New(cfg Config) *Handler {
	store := cfg.Store
	if store == nil {
		store = runners.NewPostgresStore()
	}
	return &Handler{
		BaseURL:     cfg.BaseURL,
		Registry:    cfg.Registry,
		AuthService: cfg.AuthService,
		Store:       store,
	}
}

func (h *Handler) store() runners.Store {
	if h.Store != nil {
		return h.Store
	}
	return runners.NewPostgresStore()
}
