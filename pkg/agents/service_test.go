package agents_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type blockingProvider struct {
	created  atomic.Int32
	onCreate func()
}

func (b *blockingProvider) Name() string { return testProviderName }

func (b *blockingProvider) CreateSession(_ context.Context, _ agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	b.created.Add(1)
	if b.onCreate != nil {
		b.onCreate()
	}
	return &agents.CreateSessionResult{ProviderSessionID: "provider-session-" + uuid.NewString()}, nil
}

func (b *blockingProvider) SendMessage(context.Context, string, string, agents.SendMessageOptions) error {
	return nil
}

func (b *blockingProvider) InterruptSession(context.Context, string) error { return nil }
func (b *blockingProvider) DefineOutcome(_ context.Context, _ string, _ agents.DefineOutcomeOptions) error {
	return nil
}

func (b *blockingProvider) StreamEvents(context.Context, string, func(agents.ProviderEvent) error) error {
	return nil
}

const testProviderName = "test"

type fakeProvider struct {
	mu               sync.Mutex
	createCalled     int
	sendCalled       int
	sentSessions     []string
	defineSessions   []string
	lastPreamble     string
	lastImages       []agents.MessageImage
	lastOutcomeOpts  agents.DefineOutcomeOptions
	createSessionErr error
	createHook       func() error
	sendErr          error
	sendErrs         []error
	defineOutcomeErr error
	defineErrs       []error
}

func (f *fakeProvider) Name() string { return testProviderName }

func (f *fakeProvider) CreateSession(_ context.Context, _ agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalled++
	if f.createSessionErr != nil {
		return nil, f.createSessionErr
	}
	if f.createHook != nil {
		if err := f.createHook(); err != nil {
			return nil, err
		}
	}
	return &agents.CreateSessionResult{ProviderSessionID: "provider-session-" + uuid.NewString()}, nil
}

func (f *fakeProvider) SendMessage(_ context.Context, providerSessionID string, _ string, opts agents.SendMessageOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalled++
	f.sentSessions = append(f.sentSessions, providerSessionID)
	f.lastPreamble = opts.ContextPreamble
	f.lastImages = opts.Images
	if len(f.sendErrs) > 0 {
		err := f.sendErrs[0]
		f.sendErrs = f.sendErrs[1:]
		return err
	}
	return f.sendErr
}

func (f *fakeProvider) InterruptSession(_ context.Context, _ string) error {
	return nil
}

func (f *fakeProvider) DefineOutcome(_ context.Context, providerSessionID string, opts agents.DefineOutcomeOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defineSessions = append(f.defineSessions, providerSessionID)
	if len(f.defineErrs) > 0 {
		err := f.defineErrs[0]
		f.defineErrs = f.defineErrs[1:]
		return err
	}
	if f.defineOutcomeErr != nil {
		return f.defineOutcomeErr
	}
	f.lastOutcomeOpts = opts
	return nil
}

func (f *fakeProvider) StreamEvents(_ context.Context, _ string, _ func(agents.ProviderEvent) error) error {
	return nil
}

func newService(t *testing.T, r *support.ResourceRegistry, provider agents.Provider) *agents.Service {
	t.Helper()
	signer := jwt.NewSigner("test-secret")
	return agents.NewService(provider, r.AuthService, signer, "https://api.test.local")
}

func setupCanvasForUser(t *testing.T, r *support.ResourceRegistry) *models.Canvas {
	t.Helper()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	return canvas
}

func TestService_EnsureSession_CreatesOnFirstCall(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, testProviderName, session.Provider)
	assert.Equal(t, models.AgentSessionStatusIdle, session.Status)
	assert.Equal(t, 1, provider.createCalled)
}

func TestService_EnsureSession_ConcurrentCallsProvisionOnce(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	// The provider blocks the first CreateSession until both callers have
	// reached the lock-protected region, ensuring they would race without
	// the advisory lock.
	release := make(chan struct{})
	reached := make(chan struct{}, 1)
	provider := &blockingProvider{onCreate: func() {
		select {
		case reached <- struct{}{}:
		default:
		}
		<-release
	}}
	svc := newService(t, r, provider)

	type result struct {
		session *models.AgentSession
		err     error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			s, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
			results <- result{s, err}
		}()
	}

	<-reached
	close(release)

	first := <-results
	second := <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	assert.Equal(t, first.session.ID, second.session.ID, "both callers must see the same session row")
	assert.Equal(t, int32(1), provider.created.Load(), "upstream session must be provisioned exactly once")
}

func TestService_EnsureSession_IsIdempotent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	first, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	second, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 1, provider.createCalled, "second call must not provision a new upstream session")
}

func TestService_EnsureSession_FailsWhenProviderErrors(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{createSessionErr: errors.New("provider boom")}
	svc := newService(t, r, provider)

	_, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.Error(t, err)

	var count int64
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestService_EnsureSession_DeniedWithoutPermission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	otherUser := uuid.New()
	_, err := svc.EnsureSession(context.Background(), r.Organization.ID, otherUser, canvas.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, agents.ErrSessionForbidden))
	assert.Equal(t, 0, provider.createCalled)
}

func TestService_GetSession_PrivateToUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	got, err := svc.GetSession(r.Organization.ID, r.User, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)

	_, err = svc.GetSession(r.Organization.ID, uuid.New(), session.ID)
	require.Error(t, err)
}

func TestService_SendMessage_ReturnsPersistedUserMessage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	require.NotEqual(t, uuid.Nil, persisted.ID)
	assert.Equal(t, "hello", persisted.Content)
}

func TestService_SendMessage_ForwardsAndPersistsImages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	images := []agents.MessageImage{{MediaType: "image/png", Data: "aGVsbG8="}}
	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "", images)
	require.NoError(t, err)
	require.Len(t, provider.lastImages, 1)
	assert.Equal(t, "image/png", provider.lastImages[0].MediaType)
	require.Len(t, persisted.Images, 1)
	assert.Equal(t, "aGVsbG8=", persisted.Images[0].Data)

	stored, err := svc.ListMessages(session.ID, uuid.Nil, 10)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Len(t, stored[0].Images, 1)
	assert.Equal(t, "image/png", stored[0].Images[0].MediaType)
}

func TestService_SendMessage_AllowsFollowUpWhenSessionIsStreaming(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, 1, provider.sendCalled)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_ProviderBusyKeepsSessionStreaming(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{sendErr: agents.ErrSessionBusy}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.ErrorIs(t, err, agents.ErrSessionBusy)
	require.Nil(t, persisted)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_RecreatesUnavailableProviderSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{
		sendErrs: []error{agents.ErrProviderSessionUnavailable, nil},
	}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	originalProviderSessionID := session.ProviderSessionID

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	require.Len(t, provider.sentSessions, 2)
	assert.Equal(t, originalProviderSessionID, provider.sentSessions[0])
	assert.Equal(t, refreshed.ProviderSessionID, provider.sentSessions[1])
	assert.NotEqual(t, originalProviderSessionID, refreshed.ProviderSessionID)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_ReturnsBusyWhenRecoveredProviderSessionIsBusy(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{
		sendErrs: []error{agents.ErrProviderSessionUnavailable, agents.ErrSessionBusy},
	}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.ErrorIs(t, err, agents.ErrSessionBusy)
	require.Nil(t, persisted)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	require.Len(t, provider.sentSessions, 2)
	assert.NotEqual(t, session.ProviderSessionID, refreshed.ProviderSessionID)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_DoesNotHoldSessionLockWhileCreatingRecoveredProviderSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	var sessionID uuid.UUID
	provider := &fakeProvider{
		sendErrs: []error{agents.ErrProviderSessionUnavailable, nil},
		createHook: func() error {
			if sessionID == uuid.Nil {
				return nil
			}
			return database.Conn().Transaction(func(tx *gorm.DB) error {
				_, err := models.LockAgentSessionInTransaction(tx, sessionID)
				return err
			})
		},
	}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	sessionID = session.ID

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)
}

func TestService_DefineOutcome_ReturnsBusyWhenRecoveredProviderSessionIsBusy(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{
		defineErrs: []error{agents.ErrProviderSessionUnavailable, agents.ErrSessionBusy},
	}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	err = svc.DefineOutcome(context.Background(), r.Organization.ID, r.User, session.ID, "build", "- done", 1)
	require.ErrorIs(t, err, agents.ErrSessionBusy)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	require.Len(t, provider.defineSessions, 2)
	assert.NotEqual(t, session.ProviderSessionID, refreshed.ProviderSessionID)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_RecoversFailedSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusFailed))

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "retry", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, 1, provider.sendCalled)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestService_SendMessage_RefreshesPreambleEveryTurn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "first", nil)
	require.NoError(t, err)
	assert.Contains(t, provider.lastPreamble, canvas.ID.String())
	assert.Contains(t, provider.lastPreamble, "api_token:")
	assert.Contains(t, provider.lastPreamble, "api_token_expires_at:")
	assert.Contains(t, provider.lastPreamble, "SUPERPLANE_URL=<api_base_url> SUPERPLANE_TOKEN=<api_token> superplane ...")
	assert.Contains(t, provider.lastPreamble, "Do not run `superplane version` as a preflight.")
	assert.Contains(t, provider.lastPreamble, "[Canvas Snapshot]")
	assert.Contains(t, provider.lastPreamble, "node_count:")
	assert.Contains(t, provider.lastPreamble, "  - canvases:update_version:"+canvas.ID.String())
	assert.Contains(t, provider.lastPreamble, "GET /api/v1/canvases/{canvas_id}/console")
	assert.NotContains(t, provider.lastPreamble, "  - canvases:update:"+canvas.ID.String())
	assert.NotContains(t, provider.lastPreamble, "  - canvases:publish:"+canvas.ID.String())

	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	provider.lastPreamble = "<sentinel>"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "second", nil)
	require.NoError(t, err)
	assert.Contains(t, provider.lastPreamble, "api_token:",
		"a fresh api_token must be re-injected on every turn so the session never expires mid-conversation")
}

func TestService_SendMessage_FirstTurnPreambleSurvivesProviderFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{sendErr: errors.New("provider boom")}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "first", nil)
	require.Error(t, err)

	provider.sendErr = nil
	provider.lastPreamble = "<sentinel>"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "retry", nil)
	require.NoError(t, err)
	assert.Contains(t, provider.lastPreamble, "api_token:",
		"preamble must still be injected after the previous attempt failed at the provider")
}

func TestService_DefineOutcome_RefreshesPreambleForBuildLoop(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	err = svc.DefineOutcome(
		context.Background(),
		r.Organization.ID,
		r.User,
		session.ID,
		"Build the requested workflow",
		"- Draft version created",
		3,
	)
	require.NoError(t, err)
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "SUPERPLANE_URL=<api_base_url> SUPERPLANE_TOKEN=<api_token> superplane ...")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "[Agent Mode: BUILD]")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "Prefer 'superplane_app' action 'update_draft'")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "superplane apps console set ... -f console.yaml --draft")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "ref/docs/prd/console-and-widgets.md")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "api_token:")
}

func TestService_SendMessage_PrivateToUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, uuid.New(), session.ID, "intrusion", nil)
	require.Error(t, err)
	assert.Equal(t, 0, provider.sendCalled)
}

func TestService_SendMessage_RejectsEmpty(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "", nil)
	require.Error(t, err)
	assert.Equal(t, 0, provider.sendCalled)
}

func TestService_ListMessages_TailPagination(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		_, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "m", nil)
		require.NoError(t, err)
		require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	}

	latest, err := svc.ListMessages(session.ID, uuid.Nil, 2)
	require.NoError(t, err)
	require.Len(t, latest, 2)

	older, err := svc.ListMessages(session.ID, latest[0].ID, 2)
	require.NoError(t, err)
	require.Len(t, older, 2)
	assert.True(t, older[1].CreatedAt.Before(*latest[0].CreatedAt) || older[1].ID != latest[0].ID,
		"older window must precede the anchor")

	oldest, err := svc.ListMessages(session.ID, older[0].ID, 10)
	require.NoError(t, err)
	require.Len(t, oldest, 1, "only one message remains before the second page")
}
