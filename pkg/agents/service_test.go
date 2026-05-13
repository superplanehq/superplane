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
	lastPreamble     string
	createSessionErr error
	sendErr          error
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

func (f *fakeProvider) SendMessage(_ context.Context, _ string, _ string, opts agents.SendMessageOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalled++
	f.lastPreamble = opts.ContextPreamble
	return f.sendErr
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

	persisted, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "hello")
	require.NoError(t, err)
	require.NotNil(t, persisted)
	require.NotEqual(t, uuid.Nil, persisted.ID)
	assert.Equal(t, "hello", persisted.Content)
}

func TestService_SendMessage_InjectsPreambleOnFirstTurnOnly(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "first")
	require.NoError(t, err)
	assert.Contains(t, provider.lastPreamble, canvas.ID.String())
	assert.Contains(t, provider.lastPreamble, "api_token:")

	provider.lastPreamble = "<sentinel>"
	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "second")
	require.NoError(t, err)
	assert.Equal(t, "", provider.lastPreamble)
}

func TestService_SendMessage_PrivateToUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas := setupCanvasForUser(t, r)
	provider := &fakeProvider{}
	svc := newService(t, r, provider)

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
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

	session, err := svc.EnsureSession(context.Background(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)

	_, err = svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "")
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
		_, err := svc.SendMessage(context.Background(), r.Organization.ID, r.User, session.ID, "m")
		require.NoError(t, err)
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
