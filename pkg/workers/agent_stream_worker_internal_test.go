package workers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
)

type fakeAgentTokenUsageService struct {
	enabled bool

	setupAccountCalls      []string
	setupOrganizationCalls [][2]string
}

func (s *fakeAgentTokenUsageService) Enabled() bool {
	return s.enabled
}

func (s *fakeAgentTokenUsageService) SetupAccount(_ context.Context, accountID string) (*usagepb.SetupAccountResponse, error) {
	s.setupAccountCalls = append(s.setupAccountCalls, accountID)
	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakeAgentTokenUsageService) SetupOrganization(
	_ context.Context,
	organizationID, accountID string,
	_ usage.SetupOrganizationDetails,
) (*usagepb.SetupOrganizationResponse, error) {
	s.setupOrganizationCalls = append(s.setupOrganizationCalls, [2]string{organizationID, accountID})
	return &usagepb.SetupOrganizationResponse{
		Limits: &usagepb.OrganizationLimits{RetentionWindowDays: 14},
	}, nil
}

func (s *fakeAgentTokenUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeAgentTokenUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return &usagepb.DescribeOrganizationLimitsResponse{
		Limits: &usagepb.OrganizationLimits{RetentionWindowDays: 14},
	}, nil
}

func (s *fakeAgentTokenUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeAgentTokenUsageService) CheckAccountLimits(context.Context, string, *usagepb.AccountState) (*usagepb.CheckAccountLimitsResponse, error) {
	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeAgentTokenUsageService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	return &usagepb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeAgentTokenUsageService)(nil)

func TestHandleProviderEvent_PublishesTurnUsageWhenSessionAlreadyReset(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "test",
		ProviderSessionID: "upstream-session",
		Status:            models.AgentSessionStatusIdle,
	}
	require.NoError(t, database.Conn().Create(session).Error)

	published := 0
	originalPublisher := publishAgentRunFinished
	publishAgentRunFinished = func(gotSession *models.AgentSession, evt agents.ProviderEvent) error {
		published++
		assert.Equal(t, session.ID, gotSession.ID)
		assert.Equal(t, "claude-sonnet-4-5", evt.Model)
		require.NotNil(t, evt.Usage)
		assert.Equal(t, int64(42), evt.Usage.TotalTokens)
		return nil
	}
	t.Cleanup(func() {
		publishAgentRunFinished = originalPublisher
	})

	var streamErr error
	err := handleProviderEvent(
		context.Background(),
		nil,
		session,
		agents.ProviderEvent{
			Type:  agents.ProviderEventTurnCompleted,
			Model: "claude-sonnet-4-5",
			Usage: &agents.TokenUsage{TotalTokens: 42},
		},
		func(messages.AgentSessionEventMessage) {},
		&streamErr,
		newCustomToolTurnState(),
	)

	assert.True(t, errors.Is(err, errSessionAlreadyReset))
	assert.Equal(t, 1, published)
	assert.NoError(t, streamErr)
}

func TestHandleProviderEvent_SyncsOrganizationBeforePublishingTokenUsage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "test",
		ProviderSessionID: "upstream-session",
		Status:            models.AgentSessionStatusStreaming,
	}
	require.NoError(t, database.Conn().Create(session).Error)

	usageService := &fakeAgentTokenUsageService{enabled: true}

	published := 0
	originalPublisher := publishAgentRunFinished
	publishAgentRunFinished = func(gotSession *models.AgentSession, evt agents.ProviderEvent) error {
		published++
		assert.Equal(t, session.ID, gotSession.ID)
		require.NotNil(t, evt.Usage)
		assert.Equal(t, int64(42), evt.Usage.TotalTokens)
		return nil
	}
	t.Cleanup(func() {
		publishAgentRunFinished = originalPublisher
	})

	var streamErr error
	err := handleProviderEvent(
		context.Background(),
		usageService,
		session,
		agents.ProviderEvent{
			Type:  agents.ProviderEventTurnCompleted,
			Model: "claude-sonnet-4-5",
			Usage: &agents.TokenUsage{TotalTokens: 42},
		},
		func(messages.AgentSessionEventMessage) {},
		&streamErr,
		newCustomToolTurnState(),
	)

	require.NoError(t, err)
	assert.NoError(t, streamErr)
	assert.Equal(t, 1, published)
	assert.Equal(t, []string{r.Account.ID.String()}, usageService.setupAccountCalls)
	require.Len(t, usageService.setupOrganizationCalls, 1)
	assert.Equal(t, r.Organization.ID.String(), usageService.setupOrganizationCalls[0][0])
	assert.Equal(t, r.Account.ID.String(), usageService.setupOrganizationCalls[0][1])
}

func TestHandleProviderEvent_PublishesOnlyNewCumulativeTokenUsage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := &models.AgentSession{
		OrganizationID:               r.Organization.ID,
		UserID:                       r.User,
		CanvasID:                     canvas.ID,
		Provider:                     "test",
		ProviderSessionID:            "upstream-session",
		Status:                       models.AgentSessionStatusStreaming,
		TrackedUsageInputTokens:      10,
		TrackedUsageOutputTokens:     5,
		TrackedUsageCacheReadTokens:  20,
		TrackedUsageCacheWriteTokens: 5,
		TrackedUsageTotalTokens:      40,
	}
	require.NoError(t, database.Conn().Create(session).Error)

	published := 0
	originalPublisher := publishAgentRunFinished
	publishAgentRunFinished = func(gotSession *models.AgentSession, evt agents.ProviderEvent) error {
		published++
		assert.Equal(t, session.ID, gotSession.ID)
		require.NotNil(t, evt.Usage)
		assert.Equal(t, int64(2), evt.Usage.InputTokens)
		assert.Equal(t, int64(3), evt.Usage.OutputTokens)
		assert.Equal(t, int64(4), evt.Usage.CacheReadTokens)
		assert.Equal(t, int64(6), evt.Usage.CacheWriteTokens)
		assert.Equal(t, int64(15), evt.Usage.TotalTokens)
		return nil
	}
	t.Cleanup(func() {
		publishAgentRunFinished = originalPublisher
	})

	var streamErr error
	err := handleProviderEvent(
		context.Background(),
		nil,
		session,
		agents.ProviderEvent{
			Type:  agents.ProviderEventTurnCompleted,
			Model: "claude-sonnet-4-5",
			Usage: &agents.TokenUsage{
				InputTokens:      12,
				OutputTokens:     8,
				CacheReadTokens:  24,
				CacheWriteTokens: 11,
				TotalTokens:      55,
			},
		},
		func(messages.AgentSessionEventMessage) {},
		&streamErr,
		newCustomToolTurnState(),
	)

	require.NoError(t, err)
	assert.NoError(t, streamErr)
	assert.Equal(t, 1, published)

	updated, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(12), updated.TrackedUsageInputTokens)
	assert.Equal(t, int64(8), updated.TrackedUsageOutputTokens)
	assert.Equal(t, int64(24), updated.TrackedUsageCacheReadTokens)
	assert.Equal(t, int64(11), updated.TrackedUsageCacheWriteTokens)
	assert.Equal(t, int64(55), updated.TrackedUsageTotalTokens)
}
