package approval

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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

// TODO: Add tests for role and group approval when RBAC is enabled
func TestApproval_HandleAction_Approved_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	user := &core.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user},
		},
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: metadata,
	}
	authCtx := &contexts.AuthContext{
		User: user,
	}

	ctx := core.ActionContext{
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		MetadataContext:       metadataCtx,
		ExecutionStateContext: stateCtx,
		AuthContext:           authCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, ChannelApproved, stateCtx.Channel)
}

// TODO: Add tests for role and group approval when RBAC is enabled
func TestApproval_HandleAction_Rejected_UsesCorrectChannel(t *testing.T) {
	approval := &Approval{}

	user := &core.User{ID: "test-user"}
	metadata := &Metadata{
		Result: StatePending,
		Records: []Record{
			{Index: 0, State: StatePending, Type: ItemTypeUser, User: user},
		},
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: metadata,
	}
	authCtx := &contexts.AuthContext{
		User: user,
	}

	ctx := core.ActionContext{
		Name: "reject",
		Parameters: map[string]any{
			"index":  float64(0),
			"reason": "Not approved",
		},
		MetadataContext:       metadataCtx,
		ExecutionStateContext: stateCtx,
		AuthContext:           authCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, ChannelRejected, stateCtx.Channel)
}

// TODO: Add tests for role and group approval when RBAC is enabled
func TestApproval_HandleAction_StillPending_DoesNotCallPass(t *testing.T) {
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
		Name: "approve",
		Parameters: map[string]any{
			"index": float64(0),
		},
		MetadataContext:       metadataCtx,
		ExecutionStateContext: stateCtx,
		AuthContext:           authCtx,
	}

	err := approval.HandleAction(ctx)

	assert.NoError(t, err)
	assert.False(t, stateCtx.Passed)
	assert.False(t, stateCtx.Finished)
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

// TODO: Add tests for role and group approval when RBAC is enabled
func TestApproval_Execute(t *testing.T) {
	approval := &Approval{}

	t.Run("with empty items immediately completes with approved channel", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		authCtx := &contexts.AuthContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"items": []any{},
			},
			MetadataContext:       metadataCtx,
			ExecutionStateContext: stateCtx,
			AuthContext:           authCtx,
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
			Configuration: map[string]any{
				"items": []any{
					map[string]any{
						"type": "user",
						"user": userID.String(),
					},
				},
			},
			MetadataContext:       metadataCtx,
			ExecutionStateContext: stateCtx,
			AuthContext:           authCtx,
		}

		err := approval.Execute(ctx)

		assert.NoError(t, err)
		assert.False(t, stateCtx.Passed)
		assert.False(t, stateCtx.Finished)
		assert.NotNil(t, metadataCtx.Metadata)
	})
}

// TODO: Add tests for role and group configuration validation when RBAC is enabled
func TestApproval_Configuration_Validation(t *testing.T) {
	approval := &Approval{}
	config := approval.Configuration()

	t.Run("items field is required", func(t *testing.T) {
		itemsField := config[0]
		assert.Equal(t, "items", itemsField.Name)
		assert.True(t, itemsField.Required)
	})

	t.Run("empty items list fails validation", func(t *testing.T) {
		configData := map[string]any{
			"items": []any{},
		}

		err := configuration.ValidateConfiguration(config, configData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must contain at least one item")
	})

	t.Run("missing items field fails validation", func(t *testing.T) {
		configData := map[string]any{}

		err := configuration.ValidateConfiguration(config, configData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is required")
	})

	t.Run("valid items list passes validation", func(t *testing.T) {
		configData := map[string]any{
			"items": []any{
				map[string]any{
					"type": "user",
					"user": "test-user-id",
				},
			},
		}

		err := configuration.ValidateConfiguration(config, configData)
		assert.NoError(t, err)
	})
}
