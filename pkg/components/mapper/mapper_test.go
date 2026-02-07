package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestMapper_Execute_SingleField(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{"input": "data"},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "greeting", "value": "hello"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, "default", stateCtx.Channel)
	assert.Equal(t, "mapper.executed", stateCtx.Type)
	assert.Len(t, stateCtx.Payloads, 1)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "hello", data["greeting"])
}

func TestMapper_Execute_MultipleFields(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "name", "value": "John"},
				{"name": "email", "value": "john@example.com"},
				{"name": "active", "value": "true"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "John", data["name"])
	assert.Equal(t, "john@example.com", data["email"])
	assert.Equal(t, "true", data["active"])
}

func TestMapper_Execute_KeepOnlyMappedTrue(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{
			"existing": "value",
			"other":    "data",
		},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "newField", "value": "newValue"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)

	// Only mapped fields should be in the output
	assert.Equal(t, "newValue", data["newField"])
	assert.NotContains(t, data, "existing")
	assert.NotContains(t, data, "other")
}

func TestMapper_Execute_KeepOnlyMappedFalse(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{
			"existing": "value",
			"other":    "data",
		},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "newField", "value": "newValue"},
			},
			"keepOnlyMapped": false,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)

	// Both existing and mapped fields should be in the output
	assert.Equal(t, "newValue", data["newField"])
	assert.Equal(t, "value", data["existing"])
	assert.Equal(t, "data", data["other"])
}

func TestMapper_Execute_DotNotation(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "user.profile.name", "value": "Alice"},
				{"name": "user.profile.email", "value": "alice@example.com"},
				{"name": "user.active", "value": "true"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)

	user, ok := data["user"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "true", user["active"])

	profile, ok := user["profile"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Alice", profile["name"])
	assert.Equal(t, "alice@example.com", profile["email"])
}

func TestMapper_Execute_ResolvedExpressions(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	// Simulate values already resolved by NodeConfigurationBuilder
	ctx := core.ExecutionContext{
		Data: map[string]any{},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "branch", "value": "refs/heads/main"},
				{"name": "author", "value": "john@example.com"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "refs/heads/main", data["branch"])
	assert.Equal(t, "john@example.com", data["author"])
}

func TestMapper_Execute_MetadataStorage(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "field1", "value": "value1"},
			},
			"keepOnlyMapped": true,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, metadataCtx.Metadata)

	metadata, ok := metadataCtx.Metadata.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, metadata, "fields")
	assert.Equal(t, true, metadata["keepOnlyMapped"])
}

func TestMapper_Setup_EmptyFields(t *testing.T) {
	mapper := &Mapper{}

	ctx := core.SetupContext{
		Configuration: map[string]any{
			"fields":         []map[string]any{},
			"keepOnlyMapped": true,
		},
	}

	err := mapper.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field mapping is required")
}

func TestMapper_Setup_DuplicateFieldNames(t *testing.T) {
	mapper := &Mapper{}

	ctx := core.SetupContext{
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "field1", "value": "value1"},
				{"name": "field1", "value": "value2"},
			},
			"keepOnlyMapped": true,
		},
	}

	err := mapper.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field name: field1")
}

func TestMapper_Setup_EmptyFieldName(t *testing.T) {
	mapper := &Mapper{}

	ctx := core.SetupContext{
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "", "value": "value1"},
			},
			"keepOnlyMapped": true,
		},
	}

	err := mapper.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field name cannot be empty")
}

func TestMapper_Setup_ValidConfiguration(t *testing.T) {
	mapper := &Mapper{}

	ctx := core.SetupContext{
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "field1", "value": "value1"},
				{"name": "field2", "value": "value2"},
			},
			"keepOnlyMapped": true,
		},
	}

	err := mapper.Setup(ctx)
	assert.NoError(t, err)
}

func TestMapper_Execute_NilInputData(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: nil,
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "field1", "value": "value1"},
			},
			"keepOnlyMapped": false,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "value1", data["field1"])
}

func TestMapper_Execute_DotNotationOverwritesExisting(t *testing.T) {
	mapper := &Mapper{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{
			"user": map[string]any{
				"name":  "Old Name",
				"email": "old@example.com",
			},
		},
		Configuration: map[string]any{
			"fields": []map[string]any{
				{"name": "user.name", "value": "New Name"},
			},
			"keepOnlyMapped": false,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := mapper.Execute(ctx)

	assert.NoError(t, err)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)

	user, ok := data["user"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "New Name", user["name"])
}

func Test_setNestedField(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    any
		expected map[string]any
	}{
		{
			name:     "simple key",
			path:     "name",
			value:    "Alice",
			expected: map[string]any{"name": "Alice"},
		},
		{
			name:  "nested key",
			path:  "user.name",
			value: "Alice",
			expected: map[string]any{
				"user": map[string]any{"name": "Alice"},
			},
		},
		{
			name:  "deeply nested key",
			path:  "a.b.c.d",
			value: "deep",
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": map[string]any{"d": "deep"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := map[string]any{}
			setNestedField(target, tt.path, tt.value)
			assert.Equal(t, tt.expected, target)
		})
	}
}
