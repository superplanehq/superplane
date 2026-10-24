package workers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/runner"
	hostedrunner "github.com/superplanehq/superplane/pkg/components/runner/superplane"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	creditRecoveryTrigger = "manual"
	creditRecoveryAgent   = "hosted-agent"
	creditRecoverySource  = "upstream"
)

func TestHostedCreditRecoveryRoutesEventsWithoutRepublishing(t *testing.T) {
	for _, source := range []string{"root event", "execution output"} {
		t.Run(source, func(t *testing.T) {
			f := newHostedCreditRecoveryFixture(t)
			canvas := f.publish(t, hostedCreditConfiguration(), nil, source == "execution output")
			before := f.requireNoCreditError(t, canvas)
			f.setGrant(t, 500)
			require.NoError(t, models.SuperPlaneRunnerReadinessError(f.db, f.orgID, nil))

			event := f.rootEvent(t, canvas)
			rootEventID := event.ID
			if source == "execution output" {
				f.route(t, event)
				processQueueNode(t, f.queueWorker, canvas.ID, creditRecoverySource)
				upstream := listNodeExecutionsForTest(t, canvas.ID, creditRecoverySource)
				require.Len(t, upstream, 1)
				_, err := upstream[0].Pass(map[string][]any{"default": {map[string]any{"ready": true}}})
				require.NoError(t, err)
				outputs, err := models.ListCanvasEvents(f.db, canvas.ID, creditRecoverySource, 10, nil)
				require.NoError(t, err)
				require.Len(t, outputs, 1)
				event = &outputs[0]
			}

			f.route(t, event)
			items := listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent)
			require.Len(t, items, 1, "new events must reach the hosted node after credit is granted, without another save")
			require.Equal(t, event.ID, items[0].EventID)
			require.Equal(t, rootEventID, items[0].RootEventID)
			require.Equal(t, event.RunID, items[0].RunID)
			f.requireRecoveredWithoutPublish(t, canvas, before)

			processQueueNode(t, f.queueWorker, canvas.ID, creditRecoveryAgent)
			executions := listNodeExecutionsForTest(t, canvas.ID, creditRecoveryAgent)
			require.Len(t, executions, 1)
			require.Equal(t, models.CanvasNodeExecutionStatePending, executions[0].State)
			require.Equal(t, event.ID, executions[0].EventID)
			require.Equal(t, rootEventID, executions[0].RootEventID)
			require.Equal(t, event.RunID, executions[0].RunID)
			require.Empty(t, listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent))
		})
	}
}

func TestHostedCreditRecoveryProcessesExistingBacklog(t *testing.T) {
	f := newHostedCreditRecoveryFixture(t)
	f.setGrant(t, 500)
	canvas := f.publish(t, hostedCreditConfiguration(), nil, false)
	event := f.rootEvent(t, canvas)
	f.route(t, event)
	items := listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent)
	require.Len(t, items, 1)
	queuedID := items[0].ID

	// The item exists before credit runs out and a real save marks the node as
	// errored. This is a reachable backlog, not a queue inserted past the router.
	f.setGrant(t, 0)
	f.republishAgent(t, canvas)
	before := f.requireNoCreditError(t, canvas)
	processQueueNode(t, f.queueWorker, canvas.ID, creditRecoveryAgent)
	require.Empty(t, listNodeExecutionsForTest(t, canvas.ID, creditRecoveryAgent))
	items = listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent)
	require.Len(t, items, 1)
	require.Equal(t, queuedID, items[0].ID)
	f.requireNoCreditError(t, canvas)

	f.setGrant(t, 500)
	require.True(t, pollReturnsNode(t, canvas.ID, creditRecoveryAgent), "the safety poll must discover credit-blocked backlog")
	require.NoError(t, f.queueWorker.tryProcessReadyNode(canvas.ID, creditRecoveryAgent, time.Now()))
	f.requireRecoveredWithoutPublish(t, canvas, before)
	require.Empty(t, listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent))
	executions := listNodeExecutionsForTest(t, canvas.ID, creditRecoveryAgent)
	require.Len(t, executions, 1)
	require.Equal(t, models.CanvasNodeExecutionStatePending, executions[0].State)
	require.Equal(t, event.ID, executions[0].EventID)
	require.Equal(t, event.RunID, executions[0].RunID)
}

func TestHostedCreditRecoveryKeepsIneligibleNodesBlocked(t *testing.T) {
	tests := []struct {
		name          string
		invalidConfig bool
		factoryBudget bool
		block         func(*testing.T, *hostedCreditRecoveryFixture)
		readiness     error
	}{
		{
			name:      "credit depleted again",
			block:     func(t *testing.T, f *hostedCreditRecoveryFixture) { f.setGrant(t, 0) },
			readiness: models.ErrSuperPlaneRunnerNoCredit,
		},
		{name: "invalid configuration hidden by the credit error", invalidConfig: true},
		{
			name: "subscription unavailable",
			block: func(t *testing.T, f *hostedCreditRecoveryFixture) {
				_, err := models.SetAdminOrganizationPlan(f.db, f.orgID, models.BillingPlanNone, nil)
				require.NoError(t, err)
			},
			readiness: models.ErrHostedSubscriptionRequired,
		},
		{
			name: "hosted model unavailable",
			block: func(t *testing.T, f *hostedCreditRecoveryFixture) {
				settings, err := models.GetInstallationLLMSettings(f.db)
				require.NoError(t, err)
				settings.DefaultHostedProvider, settings.DefaultHostedModel = nil, nil
				_, err = models.UpdateInstallationLLMSettings(f.db, *settings)
				require.NoError(t, err)
			},
			readiness: models.ErrSuperPlaneRunnerNoModel,
		},
		{name: "workspace budget depleted", factoryBudget: true, readiness: models.ErrSuperPlaneRunnerNoFactoryBudget},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newHostedCreditRecoveryFixture(t)
			configuration := hostedCreditConfiguration()
			if tc.invalidConfig {
				delete(configuration, "machineType")
				require.Error(t, (&hostedrunner.RunSuperPlane{}).ValidateNodeConfiguration(configuration))
			}
			var factoryID *uuid.UUID
			if tc.factoryBudget {
				factory, err := models.CreateFactory(f.db, f.orgID, support.RandomName("credit-budget"), "", "")
				require.NoError(t, err)
				zero := int64(0)
				require.NoError(t, factory.UpdateHostedSpendBudget(f.db, &zero))
				factoryID = &factory.ID
			}
			canvas := f.publish(t, configuration, factoryID, false)
			before := f.requireNoCreditError(t, canvas)
			f.setGrant(t, 500)
			if tc.block != nil {
				tc.block(t, f)
			}
			readiness := models.SuperPlaneRunnerReadinessError(f.db, f.orgID, factoryID)
			if tc.readiness != nil {
				require.ErrorIs(t, readiness, tc.readiness)
			} else {
				require.NoError(t, readiness, "the invalid configuration must be the only remaining blocker")
			}

			f.route(t, f.rootEvent(t, canvas))
			require.Empty(t, listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent))
			require.Empty(t, listNodeExecutionsForTest(t, canvas.ID, creditRecoveryAgent))
			after := f.requireNoCreditError(t, canvas)
			require.Equal(t, before.Configuration.Data(), after.Configuration.Data())
		})
	}
}

func TestHostedCreditRecoveryPreservesUnrelatedConfigurationErrors(t *testing.T) {
	f := newHostedCreditRecoveryFixture(t)
	f.setGrant(t, 500)
	configuration := hostedCreditConfiguration()
	delete(configuration, "machineType")
	canvas := f.publish(t, configuration, nil, false)
	before := f.node(t, canvas)
	require.Equal(t, models.CanvasNodeStateError, before.State)
	require.NotNil(t, before.StateReason)
	require.NotEqual(t, models.SuperPlaneRunnerNoCreditMessage, *before.StateReason)
	f.setGrant(t, 1000)
	require.NoError(t, models.SuperPlaneRunnerReadinessError(f.db, f.orgID, nil))

	f.route(t, f.rootEvent(t, canvas))
	after := f.node(t, canvas)
	require.Equal(t, before.State, after.State)
	require.Equal(t, before.StateReason, after.StateReason)
	require.Empty(t, listQueueItemsForTest(t, canvas.ID, creditRecoveryAgent))
	require.Empty(t, listNodeExecutionsForTest(t, canvas.ID, creditRecoveryAgent))
}

func TestHostedCreditRecoveryUsesTheTargetOrganizationCredit(t *testing.T) {
	f := newHostedCreditRecoveryFixture(t)
	other := f.otherOrganization(t)
	firstCanvas := f.publish(t, hostedCreditConfiguration(), nil, false)
	secondCanvas := other.publish(t, hostedCreditConfiguration(), nil, false)
	f.requireNoCreditError(t, firstCanvas)
	other.requireNoCreditError(t, secondCanvas)
	f.setGrant(t, 500)
	require.NoError(t, models.SuperPlaneRunnerReadinessError(f.db, f.orgID, nil))
	require.ErrorIs(t, models.SuperPlaneRunnerReadinessError(f.db, other.orgID, nil), models.ErrSuperPlaneRunnerNoCredit)

	f.route(t, f.rootEvent(t, firstCanvas))
	other.route(t, other.rootEvent(t, secondCanvas))
	require.Len(t, listQueueItemsForTest(t, firstCanvas.ID, creditRecoveryAgent), 1)
	require.Empty(t, listQueueItemsForTest(t, secondCanvas.ID, creditRecoveryAgent))
	other.requireNoCreditError(t, secondCanvas)
	credit, err := models.DescribeOrganizationLLMCredit(f.db, other.orgID)
	require.NoError(t, err)
	require.Zero(t, credit.RemainingMicros)
}

type hostedCreditRecoveryFixture struct {
	r           *support.ResourceRegistry
	db          *gorm.DB
	orgID       uuid.UUID
	userID      uuid.UUID
	accountID   uuid.UUID
	router      *EventRouter
	queueWorker *NodeQueueWorker
}

func newHostedCreditRecoveryFixture(t *testing.T) *hostedCreditRecoveryFixture {
	t.Helper()
	t.Setenv(models.BillingEnabledEnv, "true")
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())
	originalSettings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := models.UpdateInstallationLLMSettings(db.WithContext(context.WithoutCancel(t.Context())), *originalSettings)
		require.NoError(t, err)
	})
	provider, model := models.UsageProviderAnthropic, "claude-sonnet-4-6"
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: provider, Enabled: true, APIKey: []byte("unused-test-key"),
		AllowedModels: datatypes.JSONSlice[string]{model},
	})
	require.NoError(t, err)
	settings := *originalSettings
	settings.DefaultHostedProvider, settings.DefaultHostedModel = &provider, &model
	_, err = models.UpdateInstallationLLMSettings(db, settings)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", r.Organization.ID, models.LLMCreditGrantKindWelcome).
		Update("expires_at", time.Now().Add(-time.Minute)).Error)
	require.ErrorIs(t, models.SuperPlaneRunnerReadinessError(db, r.Organization.ID, nil), models.ErrSuperPlaneRunnerNoCredit)

	// Keep the production schema, validator, and Setup. Agent execution is a
	// tripwire because these tests end at queue admission and pending execution.
	action := &creditRecoveryRunner{RunSuperPlane: &hostedrunner.RunSuperPlane{}}
	r.Registry.Actions[models.SuperPlaneRunnerComponent] = action
	t.Cleanup(func() { require.Zero(t, action.executeCalls) })
	amqpURL, err := config.RabbitMQURL()
	require.NoError(t, err)
	return &hostedCreditRecoveryFixture{
		r: r, db: db, orgID: r.Organization.ID, userID: r.User, accountID: r.Account.ID,
		router: NewEventRouter(r.Registry, amqpURL), queueWorker: queueWorkerForTest(r),
	}
}

func hostedCreditConfiguration() map[string]any {
	return map[string]any{
		"machineType": runner.MachineTypeE1LargeAMD64,
		"steps":       []any{map[string]any{"name": "Review", "type": "prompt", "prompt": "Review the change."}},
	}
}

func (f *hostedCreditRecoveryFixture) publish(t *testing.T, configuration map[string]any, factoryID *uuid.UUID, upstream bool) *models.Canvas {
	t.Helper()
	nodes := []models.Node{
		{ID: creditRecoveryTrigger, Name: "Manual Run", Type: models.NodeTypeTrigger,
			Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}},
			Configuration: map[string]any{"templates": []any{map[string]any{"name": "Run", "payload": map[string]any{}}}}},
		{ID: creditRecoveryAgent, Name: "Hosted agent", Type: models.NodeTypeComponent,
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}}, Configuration: configuration},
	}
	edges := []models.Edge{{SourceID: creditRecoveryTrigger, TargetID: creditRecoveryAgent, Channel: "default"}}
	if upstream {
		nodes = append(nodes, models.Node{ID: creditRecoverySource, Name: "Upstream", Type: models.NodeTypeComponent,
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}, Configuration: map[string]any{}})
		edges = []models.Edge{
			{SourceID: creditRecoveryTrigger, TargetID: creditRecoverySource, Channel: "default"},
			{SourceID: creditRecoverySource, TargetID: creditRecoveryAgent, Channel: "default"},
		}
	}
	ctx := authentication.SetUserIdInMetadata(t.Context(), f.userID.String())
	response, err := canvases.CreateCanvas(ctx, f.r.Registry, f.r.Encryptor, f.r.AuthService,
		"https://example.invalid/webhooks", f.orgID, support.RandomName("credit-recovery"), "", factoryID, nodes, edges)
	require.NoError(t, err)
	id, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)
	canvas, err := models.FindCanvasInTransaction(f.db, f.orgID, id)
	require.NoError(t, err)
	return canvas
}

func (f *hostedCreditRecoveryFixture) setGrant(t *testing.T, cents int64) {
	t.Helper()
	credit, err := models.DescribeOrganizationLLMCredit(f.db, f.orgID)
	require.NoError(t, err)
	_, err = models.AdjustOrganizationLLMCreditBalance(f.db, models.CreditBalanceAdjustment{
		OrganizationID: f.orgID, Bucket: models.CreditBalanceBucketGrant,
		ExpectedMicros: credit.AdminRemainingMicros, TargetMicros: models.CentsToMicros(cents),
		Note: "Credit recovery test", ActorAccountID: &f.accountID,
	})
	require.NoError(t, err)
}

func (f *hostedCreditRecoveryFixture) rootEvent(t *testing.T, canvas *models.Canvas) *models.CanvasEvent {
	t.Helper()
	return support.EmitCanvasEventForNode(t, canvas.ID, creditRecoveryTrigger, "default", nil)
}

func (f *hostedCreditRecoveryFixture) route(t *testing.T, event *models.CanvasEvent) {
	t.Helper()
	require.NoError(t, f.router.LockAndProcessEvent(log.NewEntry(log.New()), *event, time.Now()))
	saved, err := models.FindCanvasEventInTransaction(f.db, event.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasEventStateRouted, saved.State)
	*event = *saved
}

func (f *hostedCreditRecoveryFixture) node(t *testing.T, canvas *models.Canvas) *models.CanvasNode {
	t.Helper()
	node, err := models.FindCanvasNode(f.db, canvas.ID, creditRecoveryAgent)
	require.NoError(t, err)
	return node
}

func (f *hostedCreditRecoveryFixture) requireNoCreditError(t *testing.T, canvas *models.Canvas) *models.CanvasNode {
	t.Helper()
	node := f.node(t, canvas)
	require.Equal(t, models.CanvasNodeStateError, node.State)
	require.NotNil(t, node.StateReason)
	require.Equal(t, models.SuperPlaneRunnerNoCreditMessage, *node.StateReason)
	return node
}

func (f *hostedCreditRecoveryFixture) requireRecoveredWithoutPublish(t *testing.T, canvas *models.Canvas, before *models.CanvasNode) {
	t.Helper()
	node := f.node(t, canvas)
	require.Equal(t, models.CanvasNodeStateReady, node.State)
	require.Nil(t, node.StateReason)
	require.Equal(t, before.Configuration.Data(), node.Configuration.Data())
	reloaded, err := models.FindCanvasInTransaction(f.db, f.orgID, canvas.ID)
	require.NoError(t, err)
	require.Equal(t, canvas.LiveVersionID, reloaded.LiveVersionID, "recovery must not publish a new canvas version")
}

func (f *hostedCreditRecoveryFixture) republishAgent(t *testing.T, canvas *models.Canvas) {
	t.Helper()
	configuration, err := structpb.NewStruct(f.node(t, canvas).Configuration.Data())
	require.NoError(t, err)
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		live, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
		if err != nil {
			return err
		}
		patcher := changesets.NewCanvasPatcher(tx, f.orgID, f.r.Registry, live)
		if err := patcher.ApplyChangeset(&changesets.CanvasChangeset{Changes: []*changesets.Change{{
			Type: changesets.ChangeTypeUpdateNode,
			Node: &changesets.ChangeNode{ID: creditRecoveryAgent, Name: "Hosted agent", Block: models.SuperPlaneRunnerComponent, Configuration: configuration},
		}}}); err != nil {
			return err
		}
		patched := patcher.GetVersion()
		draft, err := models.CreateCommitVersionWithSpecInTransaction(tx, canvas.ID, f.userID, "Save after credit depletion", patched.Nodes, patched.Edges)
		if err != nil {
			return err
		}
		publisher, err := changesets.NewCanvasPublisher(tx, canvas, draft, live, changesets.CanvasPublisherOptions{
			Registry: f.r.Registry, OrgID: f.orgID, Encryptor: f.r.Encryptor, AuthService: f.r.AuthService, WebhookBaseURL: "https://example.invalid/webhooks",
		})
		if err != nil {
			return err
		}
		return publisher.Publish(t.Context())
	}))
	reloaded, err := models.FindCanvasInTransaction(f.db, f.orgID, canvas.ID)
	require.NoError(t, err)
	*canvas = *reloaded
}

func (f *hostedCreditRecoveryFixture) otherOrganization(t *testing.T) *hostedCreditRecoveryFixture {
	t.Helper()
	organization, err := models.CreateOrganization(support.RandomName("other-credit-org"), "")
	require.NoError(t, err)
	account, err := models.CreateAccount("Other owner", support.RandomName("owner")+"@example.test")
	require.NoError(t, err)
	user, err := models.CreateUserInTransaction(f.db, organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, f.r.AuthService.SetupOrganization(f.db, organization.ID.String(), user.ID.String()))
	_, err = models.SetAdminOrganizationPlan(f.db, organization.ID, models.BillingPlanTrial, nil)
	require.NoError(t, err)
	other := *f
	other.orgID, other.userID, other.accountID = organization.ID, user.ID, account.ID
	require.ErrorIs(t, models.SuperPlaneRunnerReadinessError(f.db, other.orgID, nil), models.ErrSuperPlaneRunnerNoCredit)
	return &other
}

type creditRecoveryRunner struct {
	*hostedrunner.RunSuperPlane
	executeCalls int
}

func (r *creditRecoveryRunner) Execute(core.ExecutionContext) error {
	r.executeCalls++
	return fmt.Errorf("credit recovery tests must stop before agent execution")
}
