package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

// TestPanelTypeRoundTrip exercises the lowercase-string <-> proto-enum
// mapping at the wire boundary. The mapping is the only place where the
// internal `string` representation is allowed to drift from the
// `Console.Panel.Type` enum, so we lock it down on both sides.
func TestPanelTypeRoundTrip(t *testing.T) {
	cases := []struct {
		modelType string
		protoType pb.Console_Panel_Type
	}{
		{models.ConsolePanelTypeMarkdown, pb.Console_Panel_MARKDOWN},
		{models.ConsolePanelTypeNode, pb.Console_Panel_NODE},
		{models.ConsolePanelTypeNodes, pb.Console_Panel_NODES},
		{models.ConsolePanelTypeTable, pb.Console_Panel_TABLE},
		{models.ConsolePanelTypeChart, pb.Console_Panel_CHART},
		{models.ConsolePanelTypeNumber, pb.Console_Panel_NUMBER},
	}
	for _, c := range cases {
		t.Run(c.modelType, func(t *testing.T) {
			got, err := panelTypeFromModel(c.modelType)
			assert.NoError(t, err)
			assert.Equal(t, c.protoType, got)

			back, err := panelTypeToModel(c.protoType)
			assert.NoError(t, err)
			assert.Equal(t, c.modelType, back)
		})
	}
}

func TestPanelTypeFromModel_UnknownErrors(t *testing.T) {
	_, err := panelTypeFromModel("unknown-kind")
	assert.Error(t, err)
}

func TestPanelTypeToModel_UnspecifiedErrors(t *testing.T) {
	// Fail-closed: the proto3 default zero value must surface a clear error
	// instead of silently becoming `markdown`. Mirrors the behavior the
	// `UpdateConsole` handler relies on.
	_, err := panelTypeToModel(pb.Console_Panel_TYPE_UNSPECIFIED)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestPanelTypeToModel_UnknownErrors(t *testing.T) {
	// A made-up enum number (e.g. a future server's value reaching an older
	// client) must also be rejected rather than passing through.
	_, err := panelTypeToModel(pb.Console_Panel_Type(99))
	assert.Error(t, err)
}
