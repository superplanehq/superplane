package factories

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type stubGitHubIssueStateSource struct {
	stubIntakeItemSource
	repository string
	closed     map[int]bool
	err        error
	checks     int
	onCheck    func(number int)
}

func (s *stubGitHubIssueStateSource) Repository() string {
	return s.repository
}

func (s *stubGitHubIssueStateSource) IsIssueClosed(_ context.Context, number int) (bool, error) {
	s.checks++
	if s.onCheck != nil {
		s.onCheck(number)
	}
	if s.err != nil {
		return false, s.err
	}
	return s.closed[number], nil
}

func Test__SyncClosedGitHubBacklog(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	githubOrigin := models.WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createGitHubIntake := func(t *testing.T, factory *models.Factory) *models.FactoryIntake {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		return intake
	}

	createDraft := func(t *testing.T, factory *models.Factory, title string, origin models.WorkOrderOrigin) *models.FactoryWorkOrder {
		t.Helper()
		createdBy := r.User
		order, err := factory.CreateWorkOrderWithOrigin(
			database.DB(t.Context()),
			title,
			"",
			&createdBy,
			nil,
			nil,
			origin,
		)
		require.NoError(t, err)
		return order
	}

	githubDeps := func(source *stubGitHubIssueStateSource) IntakeDependencies {
		return IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return source, nil
			},
		}
	}

	t.Run("closes backlog tasks whose GitHub issues are closed", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		closedIssue := createDraft(t, factory, "Handle duplicate refunds", githubOrigin)
		createDraft(t, factory, "Still open on GitHub", models.WorkOrderOrigin{
			URL:   "https://github.com/acme/payments/issues/13",
			Label: "acme/payments#13",
		})
		createDraft(t, factory, "Not from GitHub", models.WorkOrderOrigin{
			URL:   "https://app.productive.io/12345/tasks/91",
			Label: "91",
		})

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(&stubGitHubIssueStateSource{
			repository: "acme/payments",
			closed:     map[int]bool{12: true, 13: false},
		}), orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(1), response.GetClosedCount())
		assert.Equal(t, int32(0), response.GetFailedCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), closedIssue.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultRejected, reloaded.Result)
	})

	t.Run("counts GitHub lookup failures without aborting the rest", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		createDraft(t, factory, "Fails lookup", githubOrigin)

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(&stubGitHubIssueStateSource{
			repository: "acme/payments",
			err:        errors.New("github unavailable"),
		}), orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())
		assert.Equal(t, int32(1), response.GetFailedCount())
	})

	t.Run("unconnected GitHub intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)

		_, err := SyncClosedGitHubBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return nil, errIntakeNotConnected
			},
		}, orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Connect this intake first.")
	})

	t.Run("missing GitHub intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)

		_, err := SyncClosedGitHubBacklog(ctx, IntakeDependencies{}, orgID, &pb.SyncClosedGitHubBacklogRequest{
			FactoryId: factory.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Connect this intake first.")
	})

	t.Run("skips GitHub issues from a different repository", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		other := createDraft(t, factory, "Other repo", models.WorkOrderOrigin{
			URL:   "https://github.com/other/repo/issues/4",
			Label: "other/repo#4",
		})

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(&stubGitHubIssueStateSource{
			repository: "acme/payments",
			closed:     map[int]bool{4: true},
		}), orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), other.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	})

	t.Run("counts a failed GitHub intake when another intake connects", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		createGitHubIntake(t, factory)
		createDraft(t, factory, "Still open", models.WorkOrderOrigin{
			URL:   "https://github.com/acme/payments/issues/13",
			Label: "acme/payments#13",
		})

		calls := 0
		response, err := SyncClosedGitHubBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				calls++
				if calls > 1 {
					return nil, errors.New("github unavailable")
				}
				return &stubGitHubIssueStateSource{
					repository: "acme/payments",
					closed:     map[int]bool{13: false},
				}, nil
			},
		}, orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())
		assert.Equal(t, int32(1), response.GetFailedCount())
	})

	t.Run("keeps the first connected source for a repository", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		createGitHubIntake(t, factory)
		order := createDraft(t, factory, "Handle duplicate refunds", githubOrigin)

		calls := 0
		response, err := SyncClosedGitHubBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				calls++
				closed := map[int]bool{12: calls > 1}
				return &stubGitHubIssueStateSource{
					repository: "acme/payments",
					closed:     closed,
				}, nil
			},
		}, orgID, &pb.SyncClosedGitHubBacklogRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	})

	t.Run("reuses GitHub issue lookups for duplicate origins", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		createDraft(t, factory, "First copy", githubOrigin)
		createDraft(t, factory, "Second copy", githubOrigin)
		source := &stubGitHubIssueStateSource{
			repository: "acme/payments",
			closed:     map[int]bool{12: false},
		}

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(source), orgID, &pb.SyncClosedGitHubBacklogRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())
		assert.Equal(t, 1, source.checks)
	})

	t.Run("counts a failed GitHub issue once for duplicate origins", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		createDraft(t, factory, "First copy", githubOrigin)
		createDraft(t, factory, "Second copy", githubOrigin)
		source := &stubGitHubIssueStateSource{
			repository: "acme/payments",
			err:        errors.New("github unavailable"),
		}

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(source), orgID, &pb.SyncClosedGitHubBacklogRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())
		assert.Equal(t, int32(1), response.GetFailedCount())
		assert.Equal(t, 1, source.checks)
	})

	t.Run("does not close a draft that was dispatched during the GitHub check", func(t *testing.T) {
		factory := newFactory(t)
		createGitHubIntake(t, factory)
		order := createDraft(t, factory, "Handle duplicate refunds", githubOrigin)
		source := &stubGitHubIssueStateSource{
			repository: "acme/payments",
			closed:     map[int]bool{12: true},
			onCheck: func(int) {
				require.NoError(t, order.TransitionOnDispatch(database.DB(t.Context()), &r.User))
			},
		}

		response, err := SyncClosedGitHubBacklog(ctx, githubDeps(source), orgID, &pb.SyncClosedGitHubBacklogRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), response.GetClosedCount())
		assert.Equal(t, int32(0), response.GetFailedCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	})
}
