package factories

import (
	"context"
	"errors"
	"strings"
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

type stubIntakeItemAvailabilitySource struct {
	stubIntakeItemSource
	scope     string
	urlPrefix string
	available map[string]bool
	err       error
	checks    int
	onCheck   func(id string)
}

func (s *stubIntakeItemAvailabilitySource) AvailabilityScope() string {
	return s.scope
}

func (s *stubIntakeItemAvailabilitySource) ItemIDFromOriginURL(rawURL string) (string, bool) {
	if !strings.HasPrefix(rawURL, s.urlPrefix) {
		return "", false
	}
	return strings.TrimPrefix(rawURL, s.urlPrefix), true
}

func (s *stubIntakeItemAvailabilitySource) IsItemAvailable(_ context.Context, id string) (bool, error) {
	s.checks++
	if s.onCheck != nil {
		s.onCheck(id)
	}
	if s.err != nil {
		return false, s.err
	}
	return s.available[id], nil
}

type jiraBacklogAvailabilitySource struct {
	jiraIntakeItemSource
	available map[string]bool
}

func (s *jiraBacklogAvailabilitySource) IsItemAvailable(_ context.Context, id string) (bool, error) {
	return s.available[id], nil
}

func TestRefreshBacklog(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createIntake := func(t *testing.T, factory *models.Factory, source string) {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, support.RandomName("intake"))
		_, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, source)
		require.NoError(t, err)
	}

	createDraft := func(t *testing.T, factory *models.Factory, title, originURL string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrderWithOrigin(
			database.DB(t.Context()),
			title,
			"",
			&r.User,
			nil,
			nil,
			models.WorkOrderOrigin{URL: originURL, Label: models.OriginLabelFromURL(originURL)},
		)
		require.NoError(t, err)
		return order
	}

	request := func(factory *models.Factory) *pb.RefreshBacklogRequest {
		return &pb.RefreshBacklogRequest{FactoryId: factory.ID.String()}
	}

	t.Run("archives unavailable items from every readable intake source", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceGitHubIssues)
		createIntake(t, factory, models.FactoryIntakeSourceProductiveTasks)
		githubOrder := createDraft(t, factory, "Closed issue", "https://github.com/acme/payments/issues/12")
		productiveOrder := createDraft(t, factory, "Closed task", "https://app.productive.io/123/tasks/91")
		openOrder := createDraft(t, factory, "Open issue", "https://github.com/acme/payments/issues/13")

		sources := map[string]intakeItemSource{
			models.FactoryIntakeSourceGitHubIssues: &stubIntakeItemAvailabilitySource{
				scope:     "github:acme/payments",
				urlPrefix: "https://github.com/acme/payments/issues/",
				available: map[string]bool{"12": false, "13": true},
			},
			models.FactoryIntakeSourceProductiveTasks: &stubIntakeItemAvailabilitySource{
				scope:     "productive:123:project",
				urlPrefix: "https://app.productive.io/123/tasks/",
				available: map[string]bool{"91": false},
			},
		}
		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(_ context.Context, _ *gorm.DB, intake *models.FactoryIntake) (intakeItemSource, error) {
				return sources[intake.Source], nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Equal(t, int32(2), response.GetArchivedCount())
		assert.Zero(t, response.GetFailedItemCount())
		assert.Zero(t, response.GetFailedSourceCount())

		for _, order := range []*models.FactoryWorkOrder{githubOrder, productiveOrder} {
			reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
			assert.Equal(t, models.FactoryWorkOrderResultRejected, reloaded.Result)
		}
		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), openOrder.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	})

	t.Run("counts each failed item lookup once", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceGitHubIssues)
		originURL := "https://github.com/acme/payments/issues/12"
		createDraft(t, factory, "First copy", originURL)
		createDraft(t, factory, "Second copy", originURL)
		source := &stubIntakeItemAvailabilitySource{
			scope:     "github:acme/payments",
			urlPrefix: "https://github.com/acme/payments/issues/",
			err:       errors.New("provider unavailable"),
		}

		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return source, nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Zero(t, response.GetArchivedCount())
		assert.Equal(t, int32(1), response.GetFailedItemCount())
		assert.Equal(t, 1, source.checks)
	})

	t.Run("keeps an open Productive task after it moves to another project", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceProductiveTasks)
		order := createDraft(t, factory, "Moved task", "https://app.productive.io/123/tasks/91")
		source := &stubIntakeItemAvailabilitySource{
			scope:     "productive:123:original-project",
			urlPrefix: "https://app.productive.io/123/tasks/",
			available: map[string]bool{"91": true},
		}

		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return source, nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Zero(t, response.GetArchivedCount())
		assert.Zero(t, response.GetFailedItemCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	})

	t.Run("reports a failed source while another source refreshes", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceGitHubIssues)
		createIntake(t, factory, models.FactoryIntakeSourceProductiveTasks)
		source := &stubIntakeItemAvailabilitySource{
			scope:     "github:acme/payments",
			urlPrefix: "https://github.com/acme/payments/issues/",
			available: map[string]bool{},
		}

		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(_ context.Context, _ *gorm.DB, intake *models.FactoryIntake) (intakeItemSource, error) {
				if intake.Source == models.FactoryIntakeSourceProductiveTasks {
					return nil, errors.New("not connected")
				}
				return source, nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Equal(t, int32(1), response.GetFailedSourceCount())
	})

	t.Run("archives a resolved Jira issue matched by its browse origin", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceJiraIssues)
		resolved := createDraft(t, factory, "Resolved issue", "https://acme.atlassian.net/browse/ENG-42")
		open := createDraft(t, factory, "Open issue", "https://acme.atlassian.net/browse/ENG-43")
		source := &jiraBacklogAvailabilitySource{
			jiraIntakeItemSource: jiraIntakeItemSource{
				projectKey: "ENG",
				siteURL:    "https://acme.atlassian.net",
			},
			available: map[string]bool{"ENG-42": false, "ENG-43": true},
		}

		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return source, nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Equal(t, int32(1), response.GetArchivedCount())
		assert.Zero(t, response.GetFailedItemCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), resolved.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultRejected, reloaded.Result)

		reloaded, err = factory.FindWorkOrder(database.DB(t.Context()), open.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	})

	t.Run("requires a connected readable intake", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceGitHubIssues)

		_, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return nil, errIntakeNotConnected
			},
		}, orgID, request(factory))
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Connect this intake first.")
	})

	t.Run("rejects refresh when no intake supports item lookup", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceSentryExceptions)

		_, err := RefreshBacklog(ctx, IntakeDependencies{}, orgID, request(factory))
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Add a readable intake")
	})

	t.Run("does not archive a draft dispatched during its source check", func(t *testing.T) {
		factory := newFactory(t)
		createIntake(t, factory, models.FactoryIntakeSourceGitHubIssues)
		order := createDraft(t, factory, "Dispatched issue", "https://github.com/acme/payments/issues/12")
		source := &stubIntakeItemAvailabilitySource{
			scope:     "github:acme/payments",
			urlPrefix: "https://github.com/acme/payments/issues/",
			available: map[string]bool{"12": false},
			onCheck: func(string) {
				require.NoError(t, order.TransitionOnDispatch(database.DB(t.Context()), &r.User))
			},
		}

		response, err := RefreshBacklog(ctx, IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return source, nil
			},
		}, orgID, request(factory))
		require.NoError(t, err)
		assert.Zero(t, response.GetArchivedCount())

		reloaded, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	})
}
