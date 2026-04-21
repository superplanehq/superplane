package compute

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/superplanehq/superplane/pkg/core"
)

type MockIntegrationContext struct {
	mock.Mock
	core.IntegrationContext
}

func (m *MockIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	args := m.Called()
	return args.Get(0).([]core.IntegrationSubscriptionContext), args.Error(1)
}

func (m *MockIntegrationContext) Subscribe(data any) (*uuid.UUID, error) {
	args := m.Called(data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

type MockSubscription struct {
	mock.Mock
}

func (m *MockSubscription) Configuration() any {
	return m.Called().Get(0)
}

func (m *MockSubscription) SendMessage(data any) error {
	return m.Called(data).Error(0)
}

func TestOnInstanceCreated_Execute(t *testing.T) {
	trigger := &OnInstanceCreated{}
	mockIntegration := new(MockIntegrationContext)
	mockSub := new(MockSubscription)

	payload := map[string]any{
		"eventType": "com.oraclecloud.computeapi.launchinstance.end",
		"data": map[string]any{
			"resourceId":   "id1",
			"resourceName": "test-instance",
		},
	}

	mockIntegration.On("ListSubscriptions").Return([]core.IntegrationSubscriptionContext{mockSub}, nil)
	mockSub.On("SendMessage", mock.Anything).Return(nil)

	msgCtx := core.IntegrationMessageContext{
		Message:     payload,
		Integration: mockIntegration,
	}

	err := trigger.OnIntegrationMessage(msgCtx)
	if err != nil {
		t.Errorf("OnIntegrationMessage failed: %v", err)
	}

	mockSub.AssertExpectations(t)
}

func TestOnInstanceCreated_Setup(t *testing.T) {
	trigger := &OnInstanceCreated{}
	mockIntegration := new(MockIntegrationContext)
	
	mockIntegration.On("Subscribe", "onInstanceCreated").Return(nil, nil)
	
	setupCtx := core.TriggerContext{
		Integration: mockIntegration,
	}

	err := trigger.Setup(setupCtx)
	if err != nil {
		t.Errorf("Setup failed: %v", err)
	}
	mockIntegration.AssertExpectations(t)
}
