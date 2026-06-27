package agents_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
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
	mu                   sync.Mutex
	createCalled         int
	sendCalled           int
	interruptCalled      int
	interruptedSessions  []string
	sentSessions         []string
	sentMessages         []string
	defineSessions       []string
	defineDescriptions   []string
	archivedSessions     []string
	lastPreamble         string
	lastImages           []agents.MessageImage
	lastOutcomeOpts      agents.DefineOutcomeOptions
	toolSchemaRevision   string
	onToolSchemaRevision func()
	createSessionErr     error
	createHook           func() error
	sendErr              error
	sendErrs             []error
	interruptErr         error
	archiveErr           error
	defineOutcomeErr     error
	defineErrs           []error
}

func (f *fakeProvider) Name() string { return testProviderName }

func (f *fakeProvider) ToolSchemaRevision() string {
	if f.onToolSchemaRevision != nil {
		f.onToolSchemaRevision()
	}
	if f.toolSchemaRevision == "" {
		return "test-revision"
	}
	return f.toolSchemaRevision
}

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

func (f *fakeProvider) SendMessage(_ context.Context, providerSessionID string, message string, opts agents.SendMessageOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalled++
	f.sentSessions = append(f.sentSessions, providerSessionID)
	f.sentMessages = append(f.sentMessages, message)
	f.lastPreamble = opts.ContextPreamble
	f.lastImages = opts.Images
	if len(f.sendErrs) > 0 {
		err := f.sendErrs[0]
		f.sendErrs = f.sendErrs[1:]
		return err
	}
	return f.sendErr
}

func (f *fakeProvider) InterruptSession(_ context.Context, providerSessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interruptCalled++
	f.interruptedSessions = append(f.interruptedSessions, providerSessionID)
	return f.interruptErr
}

func (f *fakeProvider) DefineOutcome(_ context.Context, providerSessionID string, opts agents.DefineOutcomeOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defineSessions = append(f.defineSessions, providerSessionID)
	f.defineDescriptions = append(f.defineDescriptions, opts.Description)
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

func (f *fakeProvider) ArchiveSession(_ context.Context, providerSessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archivedSessions = append(f.archivedSessions, providerSessionID)
	return f.archiveErr
}

func newService(t *testing.T, r *support.ResourceRegistry, provider agents.Provider) *agents.Service {
	t.Helper()
	return agents.NewService(provider, r.AuthService)
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

func TestService_EnsureSession_ReplacesIdleSessionWhenToolSchemaRevisionChanges(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{toolSchemaRevision: "revision-1"}
	svc := newService(t, r, provider)

	first, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	originalProviderSessionID := first.ProviderSessionID

	provider.toolSchemaRevision = "revision-2"
	refreshed, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	assert.Equal(t, first.ID, refreshed.ID)
	assert.Equal(t, "revision-2", refreshed.AgentToolSchemaRevision)
	assert.NotEqual(t, originalProviderSessionID, refreshed.ProviderSessionID)
	assert.Nil(t, refreshed.ContextReplayedAt)
	assert.Equal(t, []string{originalProviderSessionID}, provider.archivedSessions)
	assert.Equal(t, 2, provider.createCalled)
}

func TestService_EnsureSession_ReturnsExistingSessionWhenStaleRefreshRacesWithStreamingTurn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{toolSchemaRevision: "revision-1"}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	var revisionChecks atomic.Int32
	provider.toolSchemaRevision = "revision-2"
	provider.onToolSchemaRevision = func() {
		if revisionChecks.Add(1) != 1 {
			return
		}
		require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))
	}

	refreshed, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	assert.Equal(t, session.ID, refreshed.ID)
	assert.Equal(t, session.ProviderSessionID, refreshed.ProviderSessionID)
	assert.Equal(t, 1, provider.createCalled, "busy refresh race must not create a replacement provider session")
	assert.Empty(t, provider.archivedSessions)
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

func TestService_SendMessage_RewindsPriorMessagesAfterProviderSessionRecovery(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{
		sendErrs: []error{agents.ErrProviderSessionUnavailable, nil},
	}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
		SessionID: session.ID,
		Role:      models.AgentMessageRoleUser,
		Content:   "what changed last time?",
	}))
	require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
		SessionID: session.ID,
		Role:      models.AgentMessageRoleAssistant,
		Content:   "We inspected the draft and found a missing approval node.",
	}))
	require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
		SessionID:  session.ID,
		Role:       models.AgentMessageRoleTool,
		ToolName:   "superplane_app",
		ToolStatus: models.AgentToolStatusFinished,
		Content:    `{"canvas_yaml":"very large details are compacted"}`,
	}))

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "continue from there", nil)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	require.Len(t, provider.sentMessages, 2)
	assert.Equal(t, "continue from there", provider.sentMessages[0])
	retryMessage := provider.sentMessages[1]
	assert.Contains(t, retryMessage, "[SuperPlane conversation rewind]")
	assert.Contains(t, retryMessage, "User: what changed last time?")
	assert.Contains(t, retryMessage, "Assistant: We inspected the draft")
	assert.Contains(t, retryMessage, "Tool superplane_app finished")
	assert.Contains(t, retryMessage, "[Current user request]\ncontinue from there")

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.NotNil(t, refreshed.ContextReplayedAt)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 10)
	require.NoError(t, err)
	require.Len(t, stored, 4)
	assert.Equal(t, "continue from there", stored[3].Content)
	assert.NotContains(t, stored[3].Content, "conversation rewind")
}

func TestService_SendMessage_RewindsAfterToolSchemaRefreshAndTrimsOldMessages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{toolSchemaRevision: "revision-1"}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	originalProviderSessionID := session.ProviderSessionID
	for i := 0; i < 35; i++ {
		require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
			SessionID: session.ID,
			Role:      models.AgentMessageRoleUser,
			Content:   fmt.Sprintf("prior-message-%02d", i),
		}))
	}

	provider.toolSchemaRevision = "revision-2"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "new work", nil)
	require.NoError(t, err)

	require.Len(t, provider.sentMessages, 1)
	assert.Contains(t, provider.sentMessages[0], "[SuperPlane conversation rewind]")
	assert.NotContains(t, provider.sentMessages[0], "prior-message-00")
	assert.Contains(t, provider.sentMessages[0], "prior-message-34")
	assert.Contains(t, provider.sentMessages[0], "[Current user request]\nnew work")
	assert.Equal(t, []string{originalProviderSessionID}, provider.archivedSessions)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "revision-2", refreshed.AgentToolSchemaRevision)
	assert.NotNil(t, refreshed.ContextReplayedAt)
	assert.NotEqual(t, originalProviderSessionID, refreshed.ProviderSessionID)
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
	assert.Contains(t, provider.lastPreamble, "[Canvas Snapshot]")
	assert.Contains(t, provider.lastPreamble, "node_count:")
	assert.Contains(t, provider.lastPreamble, "  - canvases:update_version:"+canvas.ID.String())
	assert.Contains(t, provider.lastPreamble, "All SuperPlane access goes through the agent tools.")
	assert.NotContains(t, provider.lastPreamble, "  - canvases:update:"+canvas.ID.String())
	assert.NotContains(t, provider.lastPreamble, "  - canvases:publish:"+canvas.ID.String())
	// The agent must never receive a usable API/CLI credential; everything
	// goes through the server-side tools.
	assert.NotContains(t, provider.lastPreamble, "api_token:")
	assert.NotContains(t, provider.lastPreamble, "api_base_url:")
	assert.NotContains(t, provider.lastPreamble, "SUPERPLANE_TOKEN")
	assert.NotContains(t, provider.lastPreamble, "superplane version")

	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	provider.lastPreamble = "<sentinel>"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "second", nil)
	require.NoError(t, err)
	assert.Contains(t, provider.lastPreamble, canvas.ID.String(),
		"the session context must be re-injected on every turn")
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
	assert.Contains(t, provider.lastPreamble, canvas.ID.String(),
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
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "[Agent Mode: BUILD]")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "Use 'superplane_app' action 'patch_draft'")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "'update_draft' for full graph or Console draft updates")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "create_draft' when 'read' returned live/no version_id")
	assert.Contains(t, provider.lastOutcomeOpts.ContextPreamble, "ref/docs/prd/console-and-widgets.md")
	assert.NotContains(t, provider.lastOutcomeOpts.ContextPreamble, "api_token:")
	assert.NotContains(t, provider.lastOutcomeOpts.ContextPreamble, "superplane apps")
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

func TestService_InterruptSession_ResetsStreamingRowToIdle(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	require.NoError(t, svc.InterruptSession(context.Background(), r.Organization.ID, r.User, session.ID))

	assert.Equal(t, 1, provider.interruptCalled)
	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status,
		"stop button must bring the row back to idle so the UI un-gates the composer")
}

func TestService_InterruptSession_ResetsLocallyWhenProviderSessionUnavailable(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{interruptErr: agents.ErrProviderSessionUnavailable}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	require.NoError(t, svc.InterruptSession(context.Background(), r.Organization.ID, r.User, session.ID))

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status,
		"upstream-gone is logically already interrupted; local row must still reset")
}

func TestService_InterruptSession_ResetsLocallyEvenWhenProviderErrors(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{interruptErr: errors.New("anthropic 500: boom")}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	// Honor user intent: a flaky provider call must not strand the row in
	// streaming. Reconciliation happens on the next SendMessage via
	// recoverProviderSession / ErrSessionBusy handling.
	require.NoError(t, svc.InterruptSession(context.Background(), r.Organization.ID, r.User, session.ID))

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status)
}

func TestService_InterruptSession_ClosesStuckToolRows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	// Simulate a tool the worker started but never closed (e.g. worker died
	// mid-turn). The interrupt path must flip it to finished so the UI stops
	// showing "Running…" forever.
	require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
		SessionID:  session.ID,
		Role:       models.AgentMessageRoleTool,
		ToolName:   "bash",
		ToolCallID: "call-stuck",
		ToolStatus: models.AgentToolStatusStarted,
		Content:    "echo hi",
	}))

	require.NoError(t, svc.InterruptSession(context.Background(), r.Organization.ID, r.User, session.ID))

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
}
