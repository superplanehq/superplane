package compute

import (
	"testing"
	"github.com/stretchr/testify/mock"
	"github.com/superplanehq/superplane/pkg/core"
)

type MockTriggerContext struct {
	mock.Mock
}

func (m *MockTriggerContext) Emit(event, payloadType string, payloads []any) error {
	return m.Called(event, payloadType, payloads).Error(0)
}

func (m *MockTriggerContext) Integration() core.IntegrationContext {
	return m.Called().Get(0).(core.IntegrationContext)
}

type MockIntegrationContext struct {
	mock.Mock
}

func (m *MockIntegrationContext) Subscribe(name string, filter any) error {
	return m.Called(name, filter).Error(0)
}
func (m *MockIntegrationContext) Ready() {}
func (m *MockIntegrationContext) ListSubscriptions() ([]core.Subscription, error) {
	args := m.Called()
	return args.Get(0).([]core.Subscription), args.Error(1)
}

func TestOnInstanceCreated_OnIntegrationMessage(t *testing.T) {
	trigger := &OnInstanceCreated{}
	mockCtx := new(MockTriggerContext)
	
	payload := `{"eventType": "com.oraclecloud.computeapi.launchinstance.end", "data": {"resourceId": "inst1"}}`
	
	mockCtx.On("Emit", "created", "oci.instance", mock.Anything).Return(nil)
	
	err := trigger.OnIntegrationMessage(mockCtx, []byte(payload))
	if err != nil {
		t.Errorf("OnIntegrationMessage failed: %v", err)
	}
	mockCtx.AssertExpectations(t)
}

func TestOnInstanceCreated_Setup(t *testing.T) {
	trigger := &OnInstanceCreated{}
	mockCtx := new(MockTriggerContext)
	mockIntegration := new(MockIntegrationContext)
	
	mockCtx.On("Integration").Return(mockIntegration)
	mockIntegration.On("Subscribe", "onInstanceCreated", mock.Anything).Return(nil)
	
	err := trigger.Setup(mockCtx)
	if err != nil {
		t.Errorf("Setup failed: %v", err)
	}
	mockIntegration.AssertExpectations(t)
}
