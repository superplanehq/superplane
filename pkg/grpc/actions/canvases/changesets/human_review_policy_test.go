package changesets

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCanvasPublisherHumanReviewAllowsConcurrentDecision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/1", State: models.FactoryPullRequestStateOpen})
	require.NoError(t, err)
	observed, err := pr.ObserveRevision(db, "current-head")
	require.NoError(t, err)
	policy := triggerNode(models.HumanReviewTriggerNodeID, "Human review", "onRun", map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})
	policy.Metadata = models.FactoryAppTemplateMetadataFor(models.FactoryAppTemplateHumanReviewID, 1, models.ProviderGitHub)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: policy.ID, Name: policy.Name, Type: policy.Type,
		Ref: datatypes.NewJSONType(policy.Ref), Configuration: datatypes.NewJSONType(policy.Configuration),
		Metadata: datatypes.NewJSONType(policy.Metadata),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	canvas.FactoryID = &factory.ID
	live, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	policy.Configuration = map[string]any{"enabled": false, "fallbackTeam": "acme/reviewers"}
	draft, err := models.CreateCommitVersionWithSpecInTransaction(db, canvas.ID, r.User, "Disable Human review", []models.Node{policy}, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	ctx = authentication.SetUserIdInMetadata(ctx, r.User.String())
	options := canvasPublisherOptions(r)
	save := db.WithContext(ctx).Begin()
	require.NoError(t, save.Error)
	defer save.Rollback()
	var savePID int
	require.NoError(t, save.Raw("SELECT pg_backend_pid()").Scan(&savePID).Error)
	publisher, err := NewCanvasPublisher(save, canvas, draft, live, options)
	require.NoError(t, err)
	saved := make(chan error, 1)
	publicationErr := pr.WithHumanReviewPublicationLock(db.WithContext(ctx), func(conn *gorm.DB) error {
		var workerPID int
		if err := conn.Raw("SELECT pg_backend_pid()").Scan(&workerPID).Error; err != nil {
			return err
		}
		go func() {
			if err := publisher.Publish(ctx); err != nil {
				save.Rollback()
				saved <- err
				return
			}
			saved <- save.Commit().Error
		}()
		// Observe the save waiting for this worker before inserting the decision.
		assert.Eventually(t, func() bool {
			var blocked bool
			err := db.Raw("SELECT ? = ANY(pg_blocking_pids(?))", workerPID, savePID).Scan(&blocked).Error
			return err == nil && blocked
		}, 3*time.Second, 10*time.Millisecond)
		return conn.Transaction(func(tx *gorm.DB) error {
			if err := pr.LockForUpdate(tx); err != nil {
				return err
			}
			_, err := pr.NewHumanReviewDecision(tx, models.FactoryHumanReviewDecision{
				RevisionID: observed.Revision.ID, AssessedCommit: "current-head", State: models.FactoryHumanReviewStateWaitingAssessment,
			})
			return err
		})
	})
	select {
	case err := <-saved:
		require.NoError(t, err, "policy save must complete")
	case <-ctx.Done():
		t.Fatal("policy save did not finish")
	}
	require.NoError(t, publicationErr, "decision creation must complete")
	decision, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	published, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	require.Equal(t, draft.ID, published.ID)
	settings, recognized := models.HumanReviewSettingsFromNodes(published.Nodes)
	require.True(t, recognized)
	require.False(t, settings.Enabled)
}

func TestCanvasPublisherHumanReviewValidatesFallbackConnection(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		noFactory   bool
		provider    string
		connected   bool
		ready       bool
		credentials bool
		wantCode    codes.Code
		wantReason  string
	}{
		{name: "requires a factory", noFactory: true, wantCode: codes.InvalidArgument, wantReason: "must belong to a Factory"},
		{name: "requires GitHub", provider: models.ProviderBitbucket, wantCode: codes.InvalidArgument, wantReason: "requires GitHub"},
		{name: "requires a connection", wantCode: codes.FailedPrecondition, wantReason: "Connect GitHub"},
		{name: "requires a ready connection", connected: true, wantCode: codes.FailedPrecondition, wantReason: "Connect GitHub"},
		{name: "requires usable credentials", connected: true, ready: true, wantCode: codes.FailedPrecondition, wantReason: "Could not connect to GitHub"},
		{name: "requires an accessible team", connected: true, ready: true, credentials: true, wantCode: codes.FailedPrecondition, wantReason: "Cannot access fallback team acme/reviewers"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			policy := triggerNode(models.HumanReviewTriggerNodeID, "Human review", "onRun", map[string]any{"enabled": false, "fallbackTeam": "acme/reviewers"})
			policy.Metadata = models.FactoryAppTemplateMetadataFor(models.FactoryAppTemplateHumanReviewID, 1, models.ProviderGitHub)
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
			live, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			if !scenario.noFactory {
				factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
				require.NoError(t, err)
				canvas.FactoryID = &factory.ID
				require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
				config := models.FactoryOnboardingConfig{VCSProvider: scenario.provider}
				if scenario.connected {
					integration := models.NewIntegration(uuid.New(), r.Organization.ID, models.ProviderGitHub, "GitHub", nil)
					setupState := datatypes.NewJSONType(models.SetupState{})
					integration.SetupState = &setupState
					integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
						{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
						{Name: common.PropertyOwner, Value: "acme"},
						{Name: common.PropertyOwnerType, Value: common.OwnerTypeOrganization},
					})
					if scenario.ready {
						integration.State = models.IntegrationStateReady
					}
					require.NoError(t, db.Create(integration).Error)
					if scenario.credentials {
						storage := contexts.NewIntegrationSecretStorage(db, r.Encryptor, integration)
						require.NoError(t, storage.Create(core.IntegrationSecretDefinition{Name: common.SecretPAT, Value: "test-token"}))
					}
					config.VCSIntegrationID = integration.ID.String()
				}
				require.NoError(t, db.Model(factory).Update("onboarding_config", datatypes.NewJSONType(config)).Error)
			}
			policy.Configuration["enabled"] = true
			draft, err := models.CreateCommitVersionWithSpecInTransaction(db, canvas.ID, r.User, "Enable Human review", []models.Node{policy}, nil)
			require.NoError(t, err)
			options := canvasPublisherOptions(r)
			// Exercise the real GitHub client without allowing external requests.
			options.Registry, err = registry.NewRegistry(r.Encryptor, registry.HTTPOptions{BlockedHosts: []string{"api.github.com"}})
			require.NoError(t, err)
			publisher, err := NewCanvasPublisher(db, canvas, draft, live, options)
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			err = publisher.Publish(ctx)
			require.Equal(t, scenario.wantCode, grpcerrors.Code(err))
			require.ErrorContains(t, err, scenario.wantReason)
			unchanged, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			require.Equal(t, live.ID, unchanged.ID)
		})
	}
}

func TestCanvasPublisherPreservesEnabledHumanReviewWithoutConnection(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(map[bool]string{false: "saved policy", true: "internal reset"}[reset], func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			policy := triggerNode(models.HumanReviewTriggerNodeID, "Human review", "onRun", map[string]any{"enabled": !reset, "fallbackTeam": "acme/reviewers"})
			policy.Metadata = models.FactoryAppTemplateMetadataFor(models.FactoryAppTemplateHumanReviewID, 1, models.ProviderGitHub)
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: policy.ID, Name: policy.Name, Type: policy.Type,
				Ref: datatypes.NewJSONType(policy.Ref), Configuration: datatypes.NewJSONType(policy.Configuration),
				Metadata: datatypes.NewJSONType(policy.Metadata),
			}}, nil)
			canvas.FactoryID = &factory.ID
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			live, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			nodes := slices.Clone(live.Nodes)
			nodes[0].Name = "Updated Human review"
			nodes[0].Configuration = map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"}
			draft, err := models.CreateCommitVersionWithSpecInTransaction(db, canvas.ID, r.User, "Update Human review", nodes, nil)
			require.NoError(t, err)
			options := canvasPublisherOptions(r)
			options.BuiltInAutomationReset = reset
			publisher, err := NewCanvasPublisher(db, canvas, draft, live, options)
			require.NoError(t, err)
			ctx := t.Context()
			if !reset {
				ctx = authentication.SetUserIdInMetadata(ctx, r.User.String())
			}
			require.NoError(t, publisher.Publish(ctx))
			published, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			require.Equal(t, draft.ID, published.ID)
			settings, recognized := models.HumanReviewSettingsFromNodes(published.Nodes)
			require.True(t, recognized)
			require.True(t, settings.Enabled)
			require.Equal(t, "acme/reviewers", settings.FallbackTeam)
		})
	}
}
