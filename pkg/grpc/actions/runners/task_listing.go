package runners

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"slices"
	"strings"
)

// ListFleetTasks validates and paginates inspection without admission checks.
func ListFleetTasks(ctx context.Context, fleet *models.RunnerFleet, states []string, limit int32, after string) (models.TaskListPage, error) {
	valid := []string{models.RunnerTaskStatePending, models.RunnerTaskStateQueued, models.RunnerTaskStateReserved,
		models.RunnerTaskStateRunning, models.RunnerTaskStateSucceeded, models.RunnerTaskStateFailed,
		models.RunnerTaskStateCanceled, models.RunnerTaskStateLost}
	for _, state := range states {
		if !slices.Contains(valid, state) {
			return models.TaskListPage{}, grpcerrors.InvalidArgument(nil, "invalid task state")
		}
	}
	page := models.ListPage{States: states, Limit: 200}
	if limit > 0 {
		page.Limit = min(int(limit), 1000)
	}
	if strings.TrimSpace(after) != "" {
		id, err := uuid.Parse(strings.TrimSpace(after))
		if err != nil {
			return models.TaskListPage{}, grpcerrors.InvalidArgument(err, "invalid after ID")
		}
		page.AfterID = &id
	}
	result, err := fleet.ListTasks(database.DB(ctx), page)
	if err != nil {
		return models.TaskListPage{}, grpcerrors.Internal(err, "failed to list fleet tasks")
	}
	return result, nil
}
