package compute

import (
	"context"
	"testing"

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/stretchr/testify/mock"
	"github.com/superplanehq/superplane/pkg/core"
)

// Mock para ExecutionStateContext
type MockExecutionState struct {
	mock.Mock
}

func (m *MockExecutionState) IsFinished() bool { return m.Called().Bool(0) }
func (m *MockExecutionState) SetKV(key, value string) error { return m.Called(key, value).Error(0) }
func (m *MockExecutionState) Emit(channel, payloadType string, payloads []any) error {
	return m.Called(channel, payloadType, payloads).Error(0)
}
func (m *MockExecutionState) Pass() error { return m.Called().Error(0) }
func (m *MockExecutionState) Fail(reason, message string) error {
	return m.Called(reason, message).Error(0)
}

// Reutilizamos el MockClientAll anterior...
type MockClientAll struct {
	mock.Mock
}

func (m *MockClientAll) LaunchInstance(ctx context.Context, request ocicore.LaunchInstanceRequest) (ocicore.LaunchInstanceResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(ocicore.LaunchInstanceResponse), args.Error(1)
}

func (m *MockClientAll) GetInstance(ctx context.Context, request ocicore.GetInstanceRequest) (ocicore.GetInstanceResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(ocicore.GetInstanceResponse), args.Error(1)
}

func (m *MockClientAll) UpdateInstance(ctx context.Context, request ocicore.UpdateInstanceRequest) (ocicore.UpdateInstanceResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(ocicore.UpdateInstanceResponse), args.Error(1)
}

func (m *MockClientAll) TerminateInstance(ctx context.Context, request ocicore.TerminateInstanceRequest) (ocicore.TerminateInstanceResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(ocicore.TerminateInstanceResponse), args.Error(1)
}

func (m *MockClientAll) InstanceAction(ctx context.Context, request ocicore.InstanceActionRequest) (ocicore.InstanceActionResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(ocicore.InstanceActionResponse), args.Error(1)
}

// 1. TEST: Create Instance
func TestCreateInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	mockState := new(MockExecutionState)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	expectedOCID := "ocid1.instance.test"
	mockClient.On("LaunchInstance", mock.Anything, mock.Anything).Return(ocicore.LaunchInstanceResponse{
		Instance: ocicore.Instance{
			Id: common.String(expectedOCID), 
			DisplayName: common.String("test"),
			LifecycleState: ocicore.InstanceLifecycleStateRunning,
			Shape: common.String("VM.Standard.E4.Flex"),
			Region: common.String("us-ashburn-1"),
		},
	}, nil)

	mockState.On("Emit", "success", "oci.instance", mock.Anything).Return(nil)

	action := &CreateInstance{}
	err := action.Execute(core.ExecutionContext{
		Context: context.Background(),
		Configuration: map[string]interface{}{
			"compartmentId": "c1", "displayName": "test", "shape": "VM", "imageId": "i1", "subnetId": "s1",
		},
		ExecutionState: mockState,
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	mockClient.AssertExpectations(t)
	mockState.AssertExpectations(t)
}

// 2. TEST: Manage Power
func TestManageInstancePower(t *testing.T) {
	mockClient := new(MockClientAll)
	mockState := new(MockExecutionState)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	mockClient.On("InstanceAction", mock.Anything, mock.MatchedBy(func(req ocicore.InstanceActionRequest) bool {
		return *req.Action == "STOP"
	})).Return(ocicore.InstanceActionResponse{}, nil)

	mockState.On("Emit", "success", "oci.instance.action", mock.Anything).Return(nil)

	action := &ManageInstancePower{}
	err := action.Execute(core.ExecutionContext{
		Context: context.Background(),
		Configuration: map[string]interface{}{
			"id": "inst1", "action": "STOP",
		},
		ExecutionState: mockState,
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	mockClient.AssertExpectations(t)
}
