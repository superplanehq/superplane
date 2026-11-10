package runners_test

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestOrganizationFleetTaskPagination(t *testing.T) {
	s := newLifecycleSteps(t)
	s.createFleet()
	db := database.DB(t.Context())
	fleet, err := models.FindOrganizationRunnerFleet(db, s.resource.Organization.ID, "direct")
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Microsecond)
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	sort.Strings(ids)
	for _, id := range ids {
		task := &models.RunnerTask{ID: uuid.MustParse(id), OrganizationID: s.resource.Organization.ID,
			FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateSucceeded,
			QueuedAt: now, StartedAt: &now, FinishedAt: &now, PayloadCiphertext: []byte(`{}`)}
		require.NoError(t, db.Create(task).Error)
	}
	list := func(query string) *pb.ListRunnerFleetTasksResponse {
		t.Helper()
		page := &pb.ListRunnerFleetTasksResponse{}
		require.NoError(t, protojson.Unmarshal(s.request(http.MethodGet, s.path+"/tasks"+query, nil, http.StatusOK), page))
		return page
	}
	first := list("?states=succeeded&limit=2")
	require.EqualValues(t, 3, first.TotalCount)
	require.True(t, first.HasNextPage)
	require.Len(t, first.Tasks, 2)
	require.Equal(t, ids[0], first.Tasks[0].Id)
	require.Equal(t, ids[1], first.Tasks[1].Id)
	require.Equal(t, s.resource.Organization.ID.String(), first.Tasks[0].OrganizationId)
	require.Equal(t, "direct", first.Tasks[0].FleetId)
	require.Equal(t, now, first.Tasks[0].QueuedAt.AsTime())
	require.Equal(t, now, first.Tasks[0].StartedAt.AsTime())
	require.Equal(t, now, first.Tasks[0].FinishedAt.AsTime())
	last := list("?states=succeeded&limit=2&after_id=" + ids[1])
	require.EqualValues(t, 3, last.TotalCount)
	require.False(t, last.HasNextPage)
	require.Len(t, last.Tasks, 1)
	require.Equal(t, ids[2], last.Tasks[0].Id)
	empty := list("?states=failed")
	require.Empty(t, empty.Tasks)
	require.Zero(t, empty.TotalCount)
	require.Empty(t, list("?after_id="+ids[2]).Tasks)
	require.Empty(t, list("?after_id="+uuid.NewString()).Tasks)
	s.request(http.MethodGet, s.path+"/tasks?after_id=invalid", nil, http.StatusBadRequest)
	s.request(http.MethodGet, s.path+"/tasks?states=invalid", nil, http.StatusBadRequest)
}
