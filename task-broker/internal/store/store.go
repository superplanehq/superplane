package store

import (
	"context"
	"strings"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"
)

// Store persists fleets and broker-scoped tasks.
type Store interface {
	CreateFleet(ctx context.Context, f *brokermodels.Fleet) error
	DeleteFleet(ctx context.Context, id string) error
	ListFleets(ctx context.Context) ([]brokermodels.Fleet, error)
	GetFleet(ctx context.Context, id string) (*brokermodels.Fleet, error)
	FindFleetByLabels(ctx context.Context, required []string) (*brokermodels.Fleet, error)

	InsertBrokerTask(ctx context.Context, t *brokermodels.BrokerTask) error
	UpdateBrokerTaskFleetTaskID(ctx context.Context, brokerID, fleetTaskID string) error
	GetBrokerTask(ctx context.Context, brokerID string) (*brokermodels.BrokerTask, error)
	DeleteBrokerTask(ctx context.Context, brokerID string) error
}

// LabelsSubset reports whether every normalized needle exists in normalized haystack.
func LabelsSubset(haystack, needles []string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, s := range haystack {
		s = normalizeLabel(s)
		if s == "" {
			continue
		}
		set[s] = struct{}{}
	}
	for _, n := range needles {
		n = normalizeLabel(n)
		if n == "" {
			continue
		}
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}

// NormalizeLabels trims space and folds ASCII case for stable matching.
func NormalizeLabels(labels []string) []string {
	var out []string
	for _, s := range labels {
		n := normalizeLabel(s)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func normalizeLabel(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
