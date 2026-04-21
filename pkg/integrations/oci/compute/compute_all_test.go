package compute

import (
	"context"
	"testing"

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/common"
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

// 1. TEST: Create Instance
func TestCreateInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	expectedOCID := "ocid1.instance.test"
	mockClient.On("LaunchInstance", mock.Anything, mock.Anything).Return(ocicore.LaunchInstanceResponse{
		Instance: ocicore.Instance{Id: common.String(expectedOCID), DisplayName: common.String("test")},
	}, nil)

	action := &CreateInstance{}
	output, err := action.Run(core.ExecutionContext{Context: context.Background()}, map[string]interface{}{
		"compartmentId": "c1", "displayName": "test", "shape": "VM", "imageId": "i1", "subnetId": "s1",
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedOCID, output.(map[string]interface{})["id"])
}

// 2. TEST: Get Instance
func TestGetInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	expectedOCID := "ocid1.instance.test"
	mockClient.On("GetInstance", mock.Anything, mock.Anything).Return(ocicore.GetInstanceResponse{
		Instance: ocicore.Instance{
			Id: common.String(expectedOCID),
			LifecycleState: ocicore.InstanceLifecycleStateRunning,
		},
	}, nil)

	action := &GetInstance{}
	output, err := action.Run(core.ExecutionContext{Context: context.Background()}, map[string]interface{}{"id": expectedOCID})

	assert.NoError(t, err)
	assert.Equal(t, "RUNNING", output.(map[string]interface{})["state"])
}

// 3. TEST: Update Instance
func TestUpdateInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	mockClient.On("UpdateInstance", mock.Anything, mock.Anything).Return(ocicore.UpdateInstanceResponse{
		Instance: ocicore.Instance{DisplayName: common.String("new-name")},
	}, nil)

	action := &UpdateInstance{}
	output, err := action.Run(core.ExecutionContext{Context: context.Background()}, map[string]interface{}{
		"id": "ocid1", "displayName": "new-name",
	})

	assert.NoError(t, err)
	assert.Equal(t, "new-name", output.(map[string]interface{})["displayName"])
}

// 4. TEST: Manage Power
func TestManageInstancePower(t *testing.T) {
	mockClient := new(MockClientAll)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	mockClient.On("InstanceAction", mock.Anything, mock.MatchedBy(func(req ocicore.InstanceActionRequest) bool {
		return req.Action == common.String("START")
	})).Return(ocicore.InstanceActionResponse{}, nil)

	action := &ManageInstancePower{}
	_, err := action.Run(core.ExecutionContext{Context: context.Background()}, map[string]interface{}{
		"id": "ocid1", "action": "START",
	})

	assert.NoError(t, err)
}

// 5. TEST: Delete Instance
func TestDeleteInstance(t *testing.T) {
	mockClient := new(MockClientAll)
	SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mockClient, nil })

	mockClient.On("TerminateInstance", mock.Anything, mock.Anything).Return(ocicore.TerminateInstanceResponse{}, nil)

	action := &DeleteInstance{}
	_, err := action.Run(core.ExecutionContext{Context: context.Background()}, map[string]interface{}{"id": "ocid1"})

	assert.NoError(t, err)
}
