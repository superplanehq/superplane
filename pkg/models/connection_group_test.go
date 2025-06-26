package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__ConnectionGroup__CalculateFieldSet(t *testing.T) {
	user := uuid.New()
	require.NoError(t, database.TruncateTables())
	canvas, err := CreateCanvas(user, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"))
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"))
	require.NoError(t, err)

	t.Run("single field", func(t *testing.T) {
		connectionGroup, err := canvas.CreateConnectionGroup(
			"single-field-group",
			uuid.NewString(),
			[]Connection{
				{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
			},
			ConnectionGroupSpec{
				GroupBy: &ConnectionGroupBySpec{
					EmitOn: ConnectionGroupEmitOnAll,
					Fields: []ConnectionGroupByField{
						{Name: "version", Expression: "ref"},
					},
				},
			},
		)

		event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)

		fieldSet, hash, err := connectionGroup.CalculateFieldSet(event)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"version": "v1"}, fieldSet)
		h, err := crypto.SHA256ForMap(fieldSet)
		require.NoError(t, err)
		assert.Equal(t, h, hash)
	})

	t.Run("multiple fields", func(t *testing.T) {
		connectionGroup, err := canvas.CreateConnectionGroup(
			"multiple-fields-group",
			uuid.NewString(),
			[]Connection{
				{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
			},
			ConnectionGroupSpec{
				GroupBy: &ConnectionGroupBySpec{
					EmitOn: ConnectionGroupEmitOnAll,
					Fields: []ConnectionGroupByField{
						{Name: "version", Expression: "ref"},
						{Name: "type", Expression: "type"},
					},
				},
			},
		)

		event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1","type":"ce"}`), []byte(`{}`))
		require.NoError(t, err)

		fieldSet, hash, err := connectionGroup.CalculateFieldSet(event)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"version": "v1", "type": "ce"}, fieldSet)
		h, err := crypto.SHA256ForMap(fieldSet)
		require.NoError(t, err)
		assert.Equal(t, h, hash)
	})
}

func Test__ConnectionGroup__ShouldEmit(t *testing.T) {
	user := uuid.New()
	require.NoError(t, database.TruncateTables())
	canvas, err := CreateCanvas(user, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"))
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"))
	require.NoError(t, err)

	t.Run("single field, emit on all", func(t *testing.T) {
		connectionGroup, err := canvas.CreateConnectionGroup(
			"single-field-group",
			uuid.NewString(),
			[]Connection{
				{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
			},
			ConnectionGroupSpec{
				GroupBy: &ConnectionGroupBySpec{
					EmitOn: ConnectionGroupEmitOnAll,
					Fields: []ConnectionGroupByField{
						{Name: "version", Expression: "ref"},
					},
				},
			},
		)

		require.NoError(t, err)

		//
		// Create version=v1 event for source1
		//
		event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		fields := map[string]string{"version": "v1"}
		hash, _ := crypto.SHA256ForMap(fields)
		fieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, hash)
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Create version=v2 event for source1
		//
		event, err = CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v2"}`), []byte(`{}`))
		require.NoError(t, err)
		fieldsV2 := map[string]string{"version": "v2"}
		v2Hash, _ := crypto.SHA256ForMap(fieldsV2)
		fieldSetV2, err := connectionGroup.CreateFieldSet(database.Conn(), fieldsV2, v2Hash)
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Connection group should not emit anything yet, for v1 and v2
		//
		shouldEmit, err := connectionGroup.ShouldEmit(database.Conn(), fieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), fieldSetV2)
		require.NoError(t, err)
		require.False(t, shouldEmit)

		//
		// Create version=v1 event for source2
		//
		event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Now that version=v1 events from both connections came in, connection group should emit,
		// while still not emitting anything for v2.
		//
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), fieldSet)
		require.NoError(t, err)
		require.True(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), fieldSetV2)
		require.NoError(t, err)
		require.False(t, shouldEmit)
	})

	t.Run("multiple fields, emit on all", func(t *testing.T) {
		connectionGroup, err := canvas.CreateConnectionGroup(
			"multiple-fields-group",
			uuid.NewString(),
			[]Connection{
				{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
			},
			ConnectionGroupSpec{
				GroupBy: &ConnectionGroupBySpec{
					EmitOn: ConnectionGroupEmitOnAll,
					Fields: []ConnectionGroupByField{
						{Name: "version", Expression: "ref"},
						{Name: "app", Expression: "app"},
					},
				},
			},
		)

		require.NoError(t, err)

		//
		// Simulate new version=v1,app=auth event coming from source1
		//
		event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1","app":"auth"}`), []byte(`{}`))
		require.NoError(t, err)
		v1AuthFields := map[string]string{"version": "v1", "app": "auth"}
		v1AuthFieldsHash, _ := crypto.SHA256ForMap(v1AuthFields)
		v1AuthfieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), v1AuthFields, v1AuthFieldsHash)
		require.NoError(t, err)
		_, err = v1AuthfieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Simulate new version=v2,app=auth event coming from source1
		//
		event, err = CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v2","app":"auth"}`), []byte(`{}`))
		require.NoError(t, err)
		v2AuthFields := map[string]string{"version": "v2", "app": "auth"}
		v2AuthFieldsHash, _ := crypto.SHA256ForMap(v2AuthFields)
		v2AuthfieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), v2AuthFields, v2AuthFieldsHash)
		require.NoError(t, err)
		_, err = v2AuthfieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Simulate new version=v1,app=core event coming from source1
		//
		event, err = CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v2","app":"core"}`), []byte(`{}`))
		require.NoError(t, err)
		v1CoreFields := map[string]string{"version": "v2", "app": "core"}
		v1CoreFieldsHash, _ := crypto.SHA256ForMap(v1CoreFields)
		v1CoreFieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), v1CoreFields, v1CoreFieldsHash)
		require.NoError(t, err)
		_, err = v1CoreFieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Connection group should not emit anything yet, for anything
		//
		shouldEmit, err := connectionGroup.ShouldEmit(database.Conn(), v1AuthfieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), v2AuthfieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), v1CoreFieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)

		//
		// Simulate new version=v1,app=auth event coming from source2
		//
		event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1","app":"auth"}`), []byte(`{}`))
		require.NoError(t, err)
		_, err = v1AuthfieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Event should be emitted for version=v1,app=auth only
		//
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), v1AuthfieldSet)
		require.NoError(t, err)
		require.True(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), v2AuthfieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)
		shouldEmit, err = connectionGroup.ShouldEmit(database.Conn(), v1CoreFieldSet)
		require.NoError(t, err)
		require.False(t, shouldEmit)
	})
}

func Test__ConnectionGroup__Emit(t *testing.T) {
	user := uuid.New()
	require.NoError(t, database.TruncateTables())
	canvas, err := CreateCanvas(user, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"))
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"))
	require.NoError(t, err)

	connectionGroup, err := canvas.CreateConnectionGroup(
		"group1",
		uuid.NewString(),
		[]Connection{
			{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
			{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
		},
		ConnectionGroupSpec{
			GroupBy: &ConnectionGroupBySpec{
				EmitOn: ConnectionGroupEmitOnAll,
				Fields: []ConnectionGroupByField{
					{Name: "version", Expression: "ref"},
					{Name: "app", Expression: "app"},
				},
			},
		},
	)

	//
	// Simulate new version=v1,app=auth event coming from source1 and source2
	//
	event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1","app":"auth"}`), []byte(`{}`))
	require.NoError(t, err)
	v1AuthFields := map[string]string{"version": "v1", "app": "auth"}
	v1AuthFieldsHash, _ := crypto.SHA256ForMap(v1AuthFields)
	v1AuthfieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), v1AuthFields, v1AuthFieldsHash)
	require.NoError(t, err)
	_, err = v1AuthfieldSet.AttachEvent(database.Conn(), event)
	require.NoError(t, err)
	event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1","app":"auth"}`), []byte(`{}`))
	require.NoError(t, err)
	_, err = v1AuthfieldSet.AttachEvent(database.Conn(), event)

	//
	// Emit and verify the structure of the outgoing event created.
	//
	require.NoError(t, connectionGroup.Emit(database.Conn(), v1AuthfieldSet))
	rawEvent, err := FindLastEventBySourceID(connectionGroup.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"version": "v1",
		"app":     "auth",
		"events": map[string]any{
			"source-1": map[string]any{"ref": "v1", "app": "auth"},
			"source-2": map[string]any{"ref": "v1", "app": "auth"},
		},
	}, rawEvent)
}
