package agents_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

const testProviderName = "test"

type fakeProvider struct {
	mu               sync.Mutex
	createCalled     int
	sendCalled       int
	archiveCalled    int
	lastMessage      string
	lastPreamble     string
	lastArchiveID    string
	createSessionErr error
	sendErr          error
	archiveErr       error
}

func (f *fakeProvider) Name() string { return testProviderName }

func (f *fakeProvider) CreateSession(_ context.Context, _ agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalled++
	if f.createSessionErr != nil {
		return nil, f.createSessionErr
	}
	return &agents.CreateSessionResult{ProviderSessionID: "provider-session-" + uuid.NewString()}, nil
}

func (f *fakeProvider) SendMessage(_ context.Context, _ string, msg string, opts agents.SendMessageOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalled++
	f.lastMessage = msg
	f.lastPreamble = opts.ContextPreamble
	return f.sendErr
}

func (f *fakeProvider) StreamEvents(_ context.Context, _ string, _ func(agents.ProviderEvent) error) error {
	return nil
}

func (f *fakeProvider) ArchiveSession(_ context.Context, providerSessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archiveCalled++
	f.lastArchiveID = providerSessionID
	return f.archiveErr
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

func TestService_CreateSession_PersistsRowAndCallsProvider(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, testProviderName, session.Provider)
	assert.Equal(t, models.AgentSessionStatusIdle, session.Status)
	assert.Equal(t, 1, provider.createCalled)

	persisted, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, persisted.ID)
	assert.Equal(t, r.User, persisted.UserID)
}

func TestService_CreateSession_FailsWhenProviderErrors(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{createSessionErr: errors.New("provider boom")}
	svc := newService(t, r, provider)

	_, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.Error(t, err)

	var count int64
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "no session should be persisted when provider call fails")
}

func TestService_CreateSession_DeniedWithoutPermission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	otherUser := uuid.New()
	_, err := svc.CreateSession(context.Background(), r.Organization.ID, otherUser, canvas.ID)
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

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	// Owning user can fetch.
	got, err := svc.GetSession(r.Organization.ID, r.User, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)

	// Another user cannot.
	_, err = svc.GetSession(r.Organization.ID, uuid.New(), session.ID)
	require.Error(t, err, "session ownership must be enforced")
}

func TestService_SendMessage_ReturnsPersistedUserMessage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello")
	require.NoError(t, err)
	require.NotNil(t, persisted, "SendMessage must not return a nil message — gRPC serialise dereferences it")
	require.NotEqual(t, uuid.Nil, persisted.ID)
	assert.Equal(t, "hello", persisted.Content)
}

func TestService_SendMessage_PersistsUserTurnAndPublishesPreambleOnFirstTurn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello")
	require.NoError(t, err)
	assert.Equal(t, models.AgentMessageRoleUser, persisted.Role)
	assert.Equal(t, "hello", persisted.Content)
	assert.Equal(t, 1, provider.sendCalled)
	assert.Contains(t, provider.lastPreamble, canvas.ID.String(), "first-turn preamble must carry the canvas id")
	assert.Contains(t, provider.lastPreamble, "api_token:", "first-turn preamble must inject the agent token")

	msgs, err := svc.ListMessages(session.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}

func TestService_SendMessage_NoPreambleOnFollowupTurns(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "first")
	require.NoError(t, err)

	provider.lastPreamble = "<sentinel>"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "second")
	require.NoError(t, err)
	assert.Equal(t, "", provider.lastPreamble, "follow-up turns must not re-inject the preamble")
}

func TestService_SendMessage_PrivateToUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, uuid.New(), session.ID, "intrusion")
	require.Error(t, err)
	assert.Equal(t, 0, provider.sendCalled)
}

func TestService_SendMessage_RejectsEmpty(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "")
	require.Error(t, err)
	assert.Equal(t, 0, provider.sendCalled)
}

func TestService_ArchiveSession_SoftArchivesLocally(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, svc.ArchiveSession(context.Background(), r.Organization.ID, r.User, session.ID))

	persisted, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.NotNil(t, persisted.ArchivedAt, "local row must be soft-archived")
	assert.Equal(t, 1, provider.archiveCalled)

	// Listing should now exclude it.
	listed, err := svc.ListSessions(r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestService_ArchiveSession_StillArchivesLocallyWhenProviderFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{archiveErr: errors.New("provider down")}
	svc := newService(t, r, provider)

	session, err := svc.CreateSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, svc.ArchiveSession(context.Background(), r.Organization.ID, r.User, session.ID))

	persisted, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.NotNil(t, persisted.ArchivedAt)
}
