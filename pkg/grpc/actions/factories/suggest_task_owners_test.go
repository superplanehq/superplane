package factories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

type pathHistory map[string][]taskowner.Commit

func (h pathHistory) ListPathCommits(_ context.Context, repositoryPath string, _ time.Time) ([]taskowner.Commit, error) {
	return h[repositoryPath], nil
}

func TestSuggestTaskOwners(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	owner := support.CreateUser(t, r, r.Organization.ID)
	require.NotNil(t, owner.AccountID)
	require.NoError(t, models.SaveAccountLinkedAccount(
		db,
		models.NewAccountLinkedAccount(*owner.AccountID, models.ProviderGitHub, uuid.NewString(), "ada", "", ""),
	))
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, []uuid.UUID{owner.ID}, nil)
	require.NoError(t, err)

	request := func(paths ...string) *pb.SuggestTaskOwnersRequest {
		return &pb.SuggestTaskOwnersRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Paths:     paths,
		}
	}

	t.Run("no paths returns members and does not read history", func(t *testing.T) {
		withTaskOwnerHistory(t, func(context.Context, IntakeDependencies, *models.Factory, string) (taskowner.History, error) {
			t.Fatal("history must not be read without paths")
			return nil, nil
		})

		resp, err := SuggestTaskOwners(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request())
		require.NoError(t, err)
		assert.Empty(t, resp.HistoryNote)
		assert.Empty(t, resp.CodeOwners)
		require.NotEmpty(t, resp.Members)
		require.Len(t, resp.CurrentOwners, 1)
		assert.Equal(t, owner.ID.String(), resp.CurrentOwners[0].UserId)
	})

	t.Run("paths rank the member who changed them", func(t *testing.T) {
		withTaskOwnerHistory(t, func(context.Context, IntakeDependencies, *models.Factory, string) (taskowner.History, error) {
			return pathHistory{"pkg/payments": {{
				SHA:         "abc",
				AuthorLogin: "ada",
				CommittedAt: time.Now().Add(-time.Hour),
			}}}, nil
		})

		resp, err := SuggestTaskOwners(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request("./pkg/payments/", "../secret"))
		require.NoError(t, err)
		assert.Equal(t, []string{"pkg/payments"}, resp.Paths)
		require.Len(t, resp.CodeOwners, 1)
		assert.Equal(t, owner.ID.String(), resp.CodeOwners[0].UserId)
		assert.Equal(t, "ada", resp.CodeOwners[0].GithubLogin)
		assert.Equal(t, int32(1), resp.CodeOwners[0].Commits)
		assert.Empty(t, resp.HistoryNote)
	})

	t.Run("missing history returns members and a note", func(t *testing.T) {
		withTaskOwnerHistory(t, func(context.Context, IntakeDependencies, *models.Factory, string) (taskowner.History, error) {
			return nil, errors.New("github down")
		})

		resp, err := SuggestTaskOwners(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request("pkg/payments"))
		require.NoError(t, err)
		assert.Equal(t, TaskOwnerHistoryUnavailable, resp.HistoryNote)
		assert.Empty(t, resp.CodeOwners)
		require.NotEmpty(t, resp.Members)
	})

	t.Run("a closed task still returns members", func(t *testing.T) {
		require.NoError(t, db.Model(order).Update("state", models.FactoryWorkOrderStateClosed).Error)

		resp, err := SuggestTaskOwners(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request())
		require.NoError(t, err)
		require.NotEmpty(t, resp.Members)
	})
}

func withTaskOwnerHistory(t *testing.T, history func(context.Context, IntakeDependencies, *models.Factory, string) (taskowner.History, error)) {
	t.Helper()
	previous := taskOwnerHistoryFor
	taskOwnerHistoryFor = history
	t.Cleanup(func() { taskOwnerHistoryFor = previous })
}
