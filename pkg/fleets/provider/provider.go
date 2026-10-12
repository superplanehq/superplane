package provider

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/artifact"
)

type Resource struct {
	ID        string
	RunnerID  string
	FleetID   string
	State     string
	CreatedAt time.Time
}

type RunnerBootstrap struct {
	RunnerID          string
	FleetID           string
	RunnerAPIURL      string
	RegistrationToken string
	Artifact          artifact.Artifact
}

type CreateRequest struct {
	RunnerID      string
	FleetID       string
	RunnerVersion string
	Bootstrap     []byte
}

type Provider interface {
	Name() string
	List(ctx context.Context, fleetID string) ([]Resource, error)
	BuildBootstrap(request RunnerBootstrap) ([]byte, error)
	Create(ctx context.Context, request CreateRequest) (Resource, error)
	Delete(ctx context.Context, resource Resource) error
}
