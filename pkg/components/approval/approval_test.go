package approval

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestApproval_HandleAction_Approved_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	role := models.RoleOrgOwner
	group := "release-approvers"

	testCases := []struct {
		name   string
		record Record
		auth   *contexts.AuthContext
	}{
		{
			name:   "user",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeUser, User: &core.User{ID: "test-user"}},
			auth:   &contexts.AuthContext{User: &core.User{ID: "test-user"}},
		},
		{
			name:   "role",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeRole, RoleRef: &core.RoleRef{Name: role}},
			auth: &contexts.AuthContext{
				User:  &core.User{ID: "test-user"},
				Roles: map[string]*core.RoleRef{role: {Name: role}},
			},
		},
		{
			name:   "group",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeGroup, GroupRef: &core.GroupRef{Name: group}},
			auth: &contexts.AuthContext{
				User:   &core.User{ID: "test-user"},
				Groups: map[string]*core.GroupRef{group: {Name: group}},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := &Metadata{
				Result:  StatePending,
				Records: []Record{testCase.record},
			}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{
				Metadata: metadata,
			}

			ctx := core.ActionContext{
				Name: "approve",
				Parameters: map[string]any{
					"index": float64(0),
				},
				Metadata:       metadataCtx,
				ExecutionState: stateCtx,
				Auth:           testCase.auth,
			}

			err := approval.HandleAction(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Equal(t, ChannelApproved, stateCtx.Channel)
		})
	}
}

func TestApproval_HandleAction_Rejected_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	role := models.RoleOrgOwner
	group := "release-approvers"

	testCases := []struct {
		name   string
		record Record
		auth   *contexts.AuthContext
	}{
		{
			name:   "user",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeUser, User: &core.User{ID: "test-user"}},
			auth:   &contexts.AuthContext{User: &core.User{ID: "test-user"}},
		},
		{
			name:   "role",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeRole, RoleRef: &core.RoleRef{Name: role}},
			auth: &contexts.AuthContext{
				User:  &core.User{ID: "test-user"},
				Roles: map[string]*core.RoleRef{role: {Name: role}},
			},
		},
		{
			name:   "group",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeGroup, GroupRef: &core.GroupRef{Name: group}},
			auth: &contexts.AuthContext{
				User:   &core.User{ID: "test-user"},
				Groups: map[string]*core.GroupRef{group: {Name: group}},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := &Metadata{
				Result:  StatePending,
				Records: []Record{testCase.record},
			}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{
				Metadata: metadata,
			}

			ctx := core.ActionContext{
				Name: "reject",
				Parameters: map[string]any{
					"index":  float64(0),
					"reason": "Not approved",
				},
				Metadata:       metadataCtx,
				ExecutionState: stateCtx,
				Auth:           testCase.auth,
			}

			err := approval.HandleAction(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Equal(t, ChannelRejected, stateCtx.Channel)
		})
	}
}

func TestApproval_HandleAction_RejectImmediatelyFinishes(t *testing.T) {
	approval := &Approval{}

	user1 := &core.User{ID: "test-user-1"}
	user2 := &core.User{ID: "test-user-2"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user1},
			{Index: 1, State: StatePending, Type: ItemTypeUser, User: user2},
		},
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: metadata,
	}
	authCtx := &contexts.AuthContext{
		User: user1,
	}

	ctx := core.ActionContext{
		Name: "reject",
		Parameters: map[string]any{
			"index":  float64(0),
			"reason": "Nope",
		},
		Metadata:       metadataCtx,
		ExecutionState: stateCtx,
		Auth:           authCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, ChannelRejected, stateCtx.Channel)
	stored := metadataCtx.Metadata.(*Metadata)
	assert.Equal(t, StateRejected, stored.Result)
}

func TestApproval_HandleAction_StillPending_DoesNotCallPass(t *testing.T) {
	approval := &Approval{}

	role := models.RoleOrgOwner
	group := "release-approvers"

	testCases := []struct {
		name   string
		record Record
		auth   *contexts.AuthContext
	}{
		{
			name:   "user",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeUser, User: &core.User{ID: "test-user-1"}},
			auth:   &contexts.AuthContext{User: &core.User{ID: "test-user-1"}},
		},
		{
			name:   "role",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeRole, RoleRef: &core.RoleRef{Name: role}},
			auth: &contexts.AuthContext{
				User:  &core.User{ID: "test-user-1"},
				Roles: map[string]*core.RoleRef{role: {Name: role}},
			},
		},
		{
			name:   "group",
			record: Record{Index: 0, State: StatePending, Type: ItemTypeGroup, GroupRef: &core.GroupRef{Name: group}},
			auth: &contexts.AuthContext{
				User:   &core.User{ID: "test-user-1"},
				Groups: map[string]*core.GroupRef{group: {Name: group}},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := &Metadata{
				Result: StatePending,
				Records: []Record{
					testCase.record,
					{Index: 1, State: StatePending, Type: ItemTypeUser, User: &core.User{ID: "test-user-2"}},
				},
			}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{
				Metadata: metadata,
			}

			ctx := core.ActionContext{
				Name: "approve",
				Parameters: map[string]any{
					"index": float64(0),
				},
				Metadata:       metadataCtx,
				ExecutionState: stateCtx,
				Auth:           testCase.auth,
			}

			err := approval.HandleAction(ctx)

			assert.NoError(t, err)
			assert.False(t, stateCtx.Passed)
			assert.False(t, stateCtx.Finished)
		})
	}
}

func TestApproval_HandleAction_ApproveOnceAcrossAllRequirements(t *testing.T) {
	approval := &Approval{}

	user := &core.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user},
			{Index: 1, State: StatePending, Type: ItemTypeAnyone},
		},
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: metadata,
	}
	authCtx := &contexts.AuthContext{
		User: user,
	}

	//
	// Approving first requirement works
	//
	ctx := core.ActionContext{
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		Metadata:       metadataCtx,
		ExecutionState: stateCtx,
		Auth:           authCtx,
	}

	err := approval.HandleAction(ctx)
	require.NoError(t, err)

	stored := metadataCtx.Metadata.(*Metadata)
	assert.Equal(t, StateApproved, stored.Records[0].State)
	assert.Equal(t, StatePending, stored.Records[1].State)

	//
	// Approving second one does not work
	//
	ctx.Parameters["index"] = float64(1)
	err = approval.HandleAction(ctx)
	assert.ErrorContains(t, err, "user has already approved/rejected another requirement")
}

func TestApproval_HandleAction_CannotApproveRequirementAgain(t *testing.T) {
	approval := &Approval{}

	user := &core.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StateApproved, Type: ItemTypeAnyone, User: &core.User{ID: "other-user"}},
			{Index: 1, State: StatePending, Type: ItemTypeAnyone},
		},
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: metadata,
	}
	authCtx := &contexts.AuthContext{
		User: user,
	}

	//
	// Trying to approve requirement that was already approved
	//
	ctx := core.ActionContext{
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		Metadata:       metadataCtx,
		ExecutionState: stateCtx,
		Auth:           authCtx,
	}

	err := approval.HandleAction(ctx)
	assert.ErrorContains(t, err, "failed to find requirement: record at index 0 is not pending")
}

func TestMetadata_UpdateResult(t *testing.T) {
	t.Run("all approved sets approved", func(t *testing.T) {
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		user1 := &core.User{ID: "user-1"}
		user2 := &core.User{ID: "user-2"}

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
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		authCtx := &contexts.AuthContext{}

		ctx := core.ExecutionContext{
			NodeMetadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"items": []any{},
			},
			Metadata:       metadataCtx,
			ExecutionState: stateCtx,
			Auth:           authCtx,
		}

		err := approval.Execute(ctx)

		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, ChannelApproved, stateCtx.Channel)
		assert.NotNil(t, metadataCtx.Metadata)
	})

	t.Run("with items does not immediately complete", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		userID := uuid.New()
		user := &core.User{ID: userID.String()}

		authCtx := &contexts.AuthContext{
			User: user,
			Users: map[string]*core.User{
				userID.String(): user,
			},
		}

		ctx := core.ExecutionContext{
			NodeMetadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "user",
						"user": userID.String(),
					},
				},
			},
			Metadata:       metadataCtx,
			ExecutionState: stateCtx,
			Auth:           authCtx,
		}

		err := approval.Execute(ctx)

		assert.NoError(t, err)
		assert.False(t, stateCtx.Passed)
		assert.False(t, stateCtx.Finished)
		assert.NotNil(t, metadataCtx.Metadata)
	})

	t.Run("with pending items publishes approval notification", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		notificationCtx := &contexts.NotificationContext{Messages: []contexts.Notification{}}

		ctx := core.ExecutionContext{
			WorkflowID:   "workflow-1",
			NodeMetadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "anyone",
					},
				},
			},
			Metadata:       metadataCtx,
			ExecutionState: stateCtx,
			Notifications:  notificationCtx,
		}

		err := approval.Execute(ctx)
		require.NoError(t, err)
		assert.Len(t, notificationCtx.Messages, 1)
	})

	t.Run("with role and group items creates records", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		role := models.RoleOrgOwner
		group := "release-approvers"

		ctx := core.ExecutionContext{
			NodeMetadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "role",
						"role": role,
					},
					map[string]any{
						"type":  "group",
						"group": group,
					},
				},
			},
			Metadata:       metadataCtx,
			ExecutionState: stateCtx,
			Auth: &contexts.AuthContext{
				User: &core.User{ID: "test-user"},
				Users: map[string]*core.User{
					"test-user": {ID: "test-user"},
				},
				Roles: map[string]*core.RoleRef{
					role: {Name: role},
				},
				Groups: map[string]*core.GroupRef{
					group: {Name: group},
				},
			},
		}

		err := approval.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, stateCtx.Passed)
		assert.False(t, stateCtx.Finished)

		stored := metadataCtx.Metadata.(*Metadata)
		require.Len(t, stored.Records, 2)
		assert.Equal(t, ItemTypeRole, stored.Records[0].Type)
		assert.Equal(t, role, stored.Records[0].RoleRef.Name)
		assert.Equal(t, ItemTypeGroup, stored.Records[1].Type)
		assert.Equal(t, group, stored.Records[1].GroupRef.Name)
	})
}
