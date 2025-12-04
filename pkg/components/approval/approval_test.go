package approval

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/superplanehq/superplane/pkg/components"
)

type MockExecutionStateContext struct {
	mock.Mock
}

func (m *MockExecutionStateContext) Pass(outputs map[string][]any) error {
	args := m.Called(outputs)
	return args.Error(0)
}

func (m *MockExecutionStateContext) Fail(reason, message string) error {
	args := m.Called(reason, message)
	return args.Error(0)
}

func (m *MockExecutionStateContext) IsFinished() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockMetadataContext struct {
	mock.Mock
}

func (m *MockMetadataContext) Set(data any) {
	m.Called(data)
}

func (m *MockMetadataContext) Get() any {
	args := m.Called()
	return args.Get(0)
}

type MockAuthContext struct {
	mock.Mock
}

func (m *MockAuthContext) AuthenticatedUser() *components.User {
	args := m.Called()
	return args.Get(0).(*components.User)
}

func (m *MockAuthContext) HasRole(role string) (bool, error) {
	args := m.Called(role)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthContext) InGroup(group string) (bool, error) {
	args := m.Called(group)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthContext) GetUser(userID uuid.UUID) (*components.User, error) {
	args := m.Called(userID)
	return args.Get(0).(*components.User), args.Error(1)
}

func TestApproval_OutputChannels(t *testing.T) {
	approval := &Approval{}
	channels := approval.OutputChannels(nil)

	assert.Len(t, channels, 2)
	assert.Equal(t, ChannelApproved, channels[0].Name)
	assert.Equal(t, "Approved", channels[0].Label)
	assert.Equal(t, "All required actors approved", channels[0].Description)

	assert.Equal(t, ChannelRejected, channels[1].Name)
	assert.Equal(t, "Rejected", channels[1].Label)
	assert.Equal(t, "At least one actor rejected (after everyone responded)", channels[1].Description)
}

func TestApproval_HandleAction_Approved_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	user := &components.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user},
		},
	}

	mockExecStateCtx := &MockExecutionStateContext{}
	mockMetadataCtx := &MockMetadataContext{}
	mockAuthCtx := &MockAuthContext{}

	mockMetadataCtx.On("Get").Return(metadata)
	mockMetadataCtx.On("Set", mock.Anything)

	mockAuthCtx.On("AuthenticatedUser").Return(user)

	mockExecStateCtx.On("Pass", mock.MatchedBy(func(outputs map[string][]any) bool {
		_, hasApproved := outputs[ChannelApproved]
		return hasApproved
	})).Return(nil)

	ctx := components.ActionContext{
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		MetadataContext:       mockMetadataCtx,
		ExecutionStateContext: mockExecStateCtx,
		AuthContext:           mockAuthCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	mockExecStateCtx.AssertExpectations(t)
}

func TestApproval_HandleAction_Rejected_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	user := &components.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user},
		},
	}

	mockExecStateCtx := &MockExecutionStateContext{}
	mockMetadataCtx := &MockMetadataContext{}
	mockAuthCtx := &MockAuthContext{}

	mockMetadataCtx.On("Get").Return(metadata)
	mockMetadataCtx.On("Set", mock.Anything)

	mockAuthCtx.On("AuthenticatedUser").Return(user)

	mockExecStateCtx.On("Pass", mock.MatchedBy(func(outputs map[string][]any) bool {
		_, hasRejected := outputs[ChannelRejected]
		return hasRejected
	})).Return(nil)

	ctx := components.ActionContext{
		Name: "reject",
		Parameters: map[string]any{
			"index":  float64(0),
			"reason": "Not approved",
		},
		MetadataContext:       mockMetadataCtx,
		ExecutionStateContext: mockExecStateCtx,
		AuthContext:           mockAuthCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	mockExecStateCtx.AssertExpectations(t)
}

func TestApproval_HandleAction_StillPending_DoesNotCallPass(t *testing.T) {
	approval := &Approval{}

	user1 := &components.User{ID: "test-user-1"}
	user2 := &components.User{ID: "test-user-2"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user1},
			{Index: 1, State: StatePending, Type: ItemTypeUser, User: user2},
		},
	}

	mockExecStateCtx := &MockExecutionStateContext{}
	mockMetadataCtx := &MockMetadataContext{}
	mockAuthCtx := &MockAuthContext{}

	mockMetadataCtx.On("Get").Return(metadata)
	mockMetadataCtx.On("Set", mock.Anything)

	mockAuthCtx.On("AuthenticatedUser").Return(user1)

	ctx := components.ActionContext{
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		MetadataContext:       mockMetadataCtx,
		ExecutionStateContext: mockExecStateCtx,
		AuthContext:           mockAuthCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)

	mockExecStateCtx.AssertNotCalled(t, "Pass")
}

func TestApproval_Setup(t *testing.T) {
	approval := &Approval{}

	t.Run("empty items returns error", func(t *testing.T) {
		ctx := components.SetupContext{
			Configuration: map[string]any{
				"items": []any{},
			},
		}

		err := approval.Setup(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid approval configuration: no user/role/group specified")
	})

	t.Run("nil items returns error", func(t *testing.T) {
		ctx := components.SetupContext{
			Configuration: map[string]any{},
		}

		err := approval.Setup(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid approval configuration: no user/role/group specified")
	})

	t.Run("valid items succeeds", func(t *testing.T) {
		ctx := components.SetupContext{
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "role",
						"role": "admin",
					},
				},
			},
		}

		err := approval.Setup(ctx)

		assert.NoError(t, err)
	})

	t.Run("invalid configuration returns error", func(t *testing.T) {
		ctx := components.SetupContext{
			Configuration: "invalid",
		}

		err := approval.Setup(ctx)

		assert.Error(t, err)
	})
}

func TestMetadata_UpdateResult(t *testing.T) {
	t.Run("all approved sets approved", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateApproved, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StateApproved, Type: ItemTypeUser, User: user2},
			},
		}

		metadata.UpdateResult()

		assert.Equal(t, StateApproved, metadata.Result)
	})

	t.Run("one rejected sets rejected", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateApproved, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StateRejected, Type: ItemTypeUser, User: user2},
			},
		}

		metadata.UpdateResult()

		assert.Equal(t, StateRejected, metadata.Result)
	})

	t.Run("first rejected sets rejected", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateRejected, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StateApproved, Type: ItemTypeUser, User: user2},
			},
		}

		metadata.UpdateResult()

		assert.Equal(t, StateRejected, metadata.Result)
	})
}

func TestMetadata_Completed(t *testing.T) {
	t.Run("all approved returns true", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateApproved, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StateApproved, Type: ItemTypeUser, User: user2},
			},
		}

		assert.True(t, metadata.Completed())
	})

	t.Run("one pending returns false", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateApproved, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StatePending, Type: ItemTypeUser, User: user2},
			},
		}

		assert.False(t, metadata.Completed())
	})

	t.Run("mixed approved and rejected returns true", func(t *testing.T) {
		user1 := &components.User{ID: "user-1"}
		user2 := &components.User{ID: "user-2"}

		metadata := &Metadata{
			Result: StatePending,
			Records: []Record{
				{Index: 0, State: StateApproved, Type: ItemTypeUser, User: user1},
				{Index: 1, State: StateRejected, Type: ItemTypeUser, User: user2},
			},
		}

		assert.True(t, metadata.Completed())
	})
}

func TestApproval_Execute(t *testing.T) {
	approval := &Approval{}

	t.Run("with empty items immediately completes with approved channel", func(t *testing.T) {
		mockExecStateCtx := &MockExecutionStateContext{}
		mockMetadataCtx := &MockMetadataContext{}
		mockAuthCtx := &MockAuthContext{}

		mockMetadataCtx.On("Set", mock.Anything)
		mockExecStateCtx.On("Pass", mock.MatchedBy(func(outputs map[string][]any) bool {
			_, hasApproved := outputs[ChannelApproved]
			return hasApproved
		})).Return(nil)

		ctx := components.ExecutionContext{
			Configuration: map[string]any{
				"items": []any{},
			},
			MetadataContext:       mockMetadataCtx,
			ExecutionStateContext: mockExecStateCtx,
			AuthContext:           mockAuthCtx,
		}

		err := approval.Execute(ctx)

		assert.NoError(t, err)
		mockExecStateCtx.AssertExpectations(t)
		mockMetadataCtx.AssertCalled(t, "Set", mock.Anything)
	})

	t.Run("with items does not immediately complete", func(t *testing.T) {
		mockExecStateCtx := &MockExecutionStateContext{}
		mockMetadataCtx := &MockMetadataContext{}
		mockAuthCtx := &MockAuthContext{}

		userID := uuid.New()
		user := &components.User{ID: userID.String()}

		mockMetadataCtx.On("Set", mock.Anything)
		mockAuthCtx.On("GetUser", userID).Return(user, nil)

		ctx := components.ExecutionContext{
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "user",
						"user": userID.String(),
					},
				},
			},
			MetadataContext:       mockMetadataCtx,
			ExecutionStateContext: mockExecStateCtx,
			AuthContext:           mockAuthCtx,
		}

		err := approval.Execute(ctx)

		assert.NoError(t, err)
		mockExecStateCtx.AssertNotCalled(t, "Pass")
		mockMetadataCtx.AssertCalled(t, "Set", mock.Anything)
	})
}
