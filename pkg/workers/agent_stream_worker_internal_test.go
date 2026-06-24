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
