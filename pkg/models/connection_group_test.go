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
	require.NoError(t, database.TruncateTables())

	user := uuid.New()
	org, err := CreateOrganization(user, uuid.New().String(), "test")
	require.NoError(t, err)
	canvas, err := CreateCanvas(user, org.ID, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"), nil)
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"), nil)
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

func Test__ConnectionGroupFieldSet__MissingConnections(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	user := uuid.New()
	org, err := CreateOrganization(user, uuid.New().String(), "test")
	require.NoError(t, err)
	canvas, err := CreateCanvas(user, org.ID, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"), nil)
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"), nil)
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
		// Connection group should still have missing connections, for v1 and v2
		//
		missingConnections, err := fieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
		missingConnections, err = fieldSetV2.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)

		//
		// Create version=v1 event for source2
		//
		event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Now that version=v1 events from both connections came in,
		// no missing connections for v1 should be there,
		// while still having some for v2.
		//
		missingConnections, err = fieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.Empty(t, missingConnections)
		missingConnections, err = fieldSetV2.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
	})

	t.Run("new field set with same hash", func(t *testing.T) {
		connectionGroup, err := canvas.CreateConnectionGroup(
			"group1",
			uuid.NewString(),
			[]Connection{
				{SourceID: source1.ID, SourceName: source1.Name, SourceType: SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: SourceTypeEventSource},
			},
			ConnectionGroupSpec{
				GroupBy: &ConnectionGroupBySpec{
					Fields: []ConnectionGroupByField{
						{Name: "version", Expression: "ref"},
					},
				},
			},
		)

		require.NoError(t, err)

		//
		// Create version=v1 event for source1 and source2
		//
		event, err := CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		fields := map[string]string{"version": "v1"}
		hash, _ := crypto.SHA256ForMap(fields)
		fieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, hash)
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)
		event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Verify that we should emit for the version=v1 field set,
		// and move the field set to processed - simulating what the worker does in that case.
		//
		missingConnections, err := fieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.Empty(t, missingConnections)
		require.NoError(t, fieldSet.UpdateState(database.Conn(), ConnectionGroupFieldSetStateProcessed, ConnectionGroupFieldSetStateReasonOK))

		//
		// Now, we send a new version=v1 event for source1 only.
		//
		event, err = CreateEvent(source1.ID, source1.Name, SourceTypeEventSource, []byte(`{"ref":"v1"}`), []byte(`{}`))
		require.NoError(t, err)
		newFieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, hash)
		require.NoError(t, err)
		_, err = newFieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Verify we shouldn't emit anything for it yet.
		//
		missingConnections, err = newFieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
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
		// All the field sets should have missing connections so far.
		//
		missingConnections, err := v1AuthfieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
		missingConnections, err = v2AuthfieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
		missingConnections, err = v1CoreFieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)

		//
		// Simulate new version=v1,app=auth event coming from source2
		//
		event, err = CreateEvent(source2.ID, source2.Name, SourceTypeEventSource, []byte(`{"ref":"v1","app":"auth"}`), []byte(`{}`))
		require.NoError(t, err)
		_, err = v1AuthfieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		//
		// Field set for version=v1,app=auth should not have missing connections.
		//
		missingConnections, err = v1AuthfieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.Empty(t, missingConnections)
		missingConnections, err = v2AuthfieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
		missingConnections, err = v1CoreFieldSet.MissingConnections(database.Conn(), connectionGroup)
		require.NoError(t, err)
		require.NotEmpty(t, missingConnections)
	})
}

func Test__ConnectionGroup__Emit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	user := uuid.New()
	org, err := CreateOrganization(user, uuid.New().String(), "test")
	require.NoError(t, err)
	canvas, err := CreateCanvas(user, org.ID, "test")
	require.NoError(t, err)
	source1, err := canvas.CreateEventSource("source-1", []byte("my-key"), nil)
	require.NoError(t, err)
	source2, err := canvas.CreateEventSource("source-2", []byte("my-key"), nil)
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
	require.NoError(t, connectionGroup.Emit(v1AuthfieldSet, ConnectionGroupFieldSetStateReasonOK, []Connection{}))
	rawEvent, err := FindLastEventBySourceID(connectionGroup.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"fields": map[string]any{
			"app":     "auth",
			"version": "v1",
		},
		"events": map[string]any{
			"source-1": map[string]any{"ref": "v1", "app": "auth"},
			"source-2": map[string]any{"ref": "v1", "app": "auth"},
		},
	}, rawEvent)
}
