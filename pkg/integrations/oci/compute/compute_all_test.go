package compute

import (
	"context"
	"testing"

	ocicommon "github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/superplanehq/superplane/pkg/core"
)

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

type MockExecutionState struct {
	mock.Mock
}

func (m *MockExecutionState) IsFinished() bool                  { return false }
func (m *MockExecutionState) SetKV(key, value string) error     { return nil }
func (m *MockExecutionState) Pass() error                       { return nil }
func (m *MockExecutionState) Fail(reason, message string) error { return nil }
func (m *MockExecutionState) Emit(channel, payloadType string, payloads []any) error {
	args := m.Called(channel, payloadType, payloads)
	return args.Error(0)
}

func TestCreateInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	mockState := new(MockExecutionState)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	mockClient.On("LaunchInstance", mock.Anything, mock.Anything).Return(ocicore.LaunchInstanceResponse{
		Instance: ocicore.Instance{Id: ocicommon.String("id1"), DisplayName: ocicommon.String("test")},
	}, nil)
	mockState.On("Emit", "default", "instance", mock.Anything).Return(nil)

	action := &CreateInstance{}
	err := action.Execute(core.ExecutionContext{
		Data: map[string]any{
			"compartmentId": "c1",
			"displayName":   "test",
			"shape":         "VM",
		},
		ExecutionState: mockState,
	})
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockState.AssertExpectations(t)
}
