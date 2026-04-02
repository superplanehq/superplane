package actions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	canvaspb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
)

func TestConfigurationFieldToProto(t *testing.T) {
	t.Run("roundtrip string default value does not introduce extra quotes", func(t *testing.T) {
		original := "https://example.com/webhook"

		field := configuration.Field{
			Name:    "url",
			Label:   "Webhook URL",
			Type:    configuration.FieldTypeString,
			Default: original,
		}

		// First roundtrip
		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.DefaultValue, "expected DefaultValue to be set")

		field2 := ProtoToConfigurationField(pbField)
		got1, ok := field2.Default.(string)
		require.True(t, ok, "expected Default to be string after first roundtrip")
		assert.Equal(t, original, got1)

		// Second roundtrip to ensure we don't accumulate quotes
		pbField2 := ConfigurationFieldToProto(field2)
		require.NotNil(t, pbField2.DefaultValue, "expected DefaultValue to be set on second roundtrip")

		field3 := ProtoToConfigurationField(pbField2)
		got2, ok := field3.Default.(string)
		require.True(t, ok, "expected Default to be string after second roundtrip")
		assert.Equal(t, original, got2)
	})

	t.Run("roundtrip non-string default value works correctly", func(t *testing.T) {
		original := []string{"monday", "wednesday"}

		field := configuration.Field{
			Name:    "days",
			Label:   "Days",
			Type:    configuration.FieldTypeList,
			Default: original,
		}

		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.DefaultValue, "expected DefaultValue to be set")

		field2 := ProtoToConfigurationField(pbField)

		got, ok := field2.Default.([]any)
		require.True(t, ok, "expected Default to be slice after roundtrip")
		require.Len(t, got, len(original))

		for i, v := range got {
			assert.Equal(t, original[i], v)
		}
	})

	t.Run("roundtrip list type options with MaxItems preserves field", func(t *testing.T) {
		maxItems := 4

		field := configuration.Field{
			Name:  "buttons",
			Label: "Buttons",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Button",
					MaxItems:  &maxItems,
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		}

		// Convert to proto
		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.TypeOptions, "expected TypeOptions to be set")
		require.NotNil(t, pbField.TypeOptions.List, "expected List options to be set")
		require.NotNil(t, pbField.TypeOptions.List.MaxItems, "expected MaxItems to be set in proto")
		assert.Equal(t, int32(maxItems), *pbField.TypeOptions.List.MaxItems)

		// Convert back from proto
		field2 := ProtoToConfigurationField(pbField)
		require.NotNil(t, field2.TypeOptions, "expected TypeOptions to be set after roundtrip")
		require.NotNil(t, field2.TypeOptions.List, "expected List options to be set after roundtrip")
		require.NotNil(t, field2.TypeOptions.List.MaxItems, "expected MaxItems to be set after roundtrip")
		assert.Equal(t, maxItems, *field2.TypeOptions.List.MaxItems)
	})
}

// TestReproduceFirstGroupWidgetConfigLoss replicates the exact bug scenario:
// YAML with 4 group widgets → CLI parsing (YAML→JSON→struct) → protojson
// (simulating gRPC gateway) → server ProtoToNodes → DB roundtrip → NodesToProto response.
// The bug: the first group widget loses its configuration.
func TestReproduceFirstGroupWidgetConfigLoss(t *testing.T) {
	// This is the YAML the user would write, matching the bug report:
	// 4 group widgets, each with configuration containing label, color, childNodeIds.
	canvasYAML := `
apiVersion: v1
kind: Canvas
metadata:
  id: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
  name: test-groups
spec:
  nodes:
    - id: group-1
      name: Group One
      type: TYPE_WIDGET
      widget:
        name: group
      configuration:
        label: First Group
        color: blue
        childNodeIds:
          - trigger-1
      position:
        x: 0
        "y": 0
    - id: group-2
      name: Group Two
      type: TYPE_WIDGET
      widget:
        name: group
      configuration:
        label: Second Group
        color: green
        childNodeIds:
          - filter-1
      position:
        x: 0
        "y": 200
    - id: group-3
      name: Group Three
      type: TYPE_WIDGET
      widget:
        name: group
      configuration:
        label: Third Group
        color: purple
        childNodeIds: []
      position:
        x: 0
        "y": 400
    - id: group-4
      name: Group Four
      type: TYPE_WIDGET
      widget:
        name: group
      configuration:
        label: Fourth Group
        color: orange
        childNodeIds:
          - trigger-1
          - filter-1
      position:
        x: 0
        "y": 600
    - id: trigger-1
      name: my_trigger
      type: TYPE_TRIGGER
      trigger:
        name: start
      configuration: {}
      position:
        x: 100
        "y": 0
    - id: filter-1
      name: my_filter
      type: TYPE_COMPONENT
      component:
        name: noop
      configuration: {}
      position:
        x: 100
        "y": 200
  edges:
    - sourceId: trigger-1
      targetId: filter-1
      channel: default
`

	// === Step 1: CLI parsing (same as pkg/cli/commands/canvases/models.ParseCanvas) ===
	var yamlObject any
	err := yaml.Unmarshal([]byte(canvasYAML), &yamlObject)
	require.NoError(t, err, "YAML unmarshal failed")

	jsonData, err := json.Marshal(yamlObject)
	require.NoError(t, err, "YAML→JSON marshal failed")

	// Inspect what the CLI would send as the JSON body
	t.Logf("CLI JSON body:\n%s", string(jsonData))

	// Parse into the OpenAPI-style structure (map-based, same types CLI uses)
	var parsed map[string]any
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	spec, _ := parsed["spec"].(map[string]any)
	require.NotNil(t, spec)
	rawNodes, _ := spec["nodes"].([]any)
	require.Len(t, rawNodes, 6, "expected 6 nodes in parsed YAML")

	// Check that the CLI-side JSON has configuration for all groups
	for i, raw := range rawNodes {
		node, _ := raw.(map[string]any)
		nodeType, _ := node["type"].(string)
		if nodeType != "TYPE_WIDGET" {
			continue
		}
		config, hasConfig := node["configuration"]
		assert.True(t, hasConfig, "CLI JSON: node %d (%s) should have configuration key", i, node["id"])
		assert.NotNil(t, config, "CLI JSON: node %d (%s) configuration should not be nil", i, node["id"])
		t.Logf("CLI JSON node %d (%s) configuration: %v", i, node["id"], config)
	}

	// === Step 2: Simulate gRPC gateway (protojson unmarshal from JSON into proto) ===
	// The gRPC gateway only receives {canvas: {metadata, spec}, ...} — not apiVersion/kind.
	// Build the request body matching the UpdateCanvasVersionRequest proto.
	requestBody := map[string]any{
		"canvas": map[string]any{
			"metadata": parsed["metadata"],
			"spec":     parsed["spec"],
		},
	}
	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)
	t.Logf("gRPC gateway request JSON:\n%s", string(requestJSON))

	var protoReq canvaspb.UpdateCanvasVersionRequest
	err = protojson.Unmarshal(requestJSON, &protoReq)
	require.NoError(t, err, "protojson unmarshal failed")

	require.NotNil(t, protoReq.Canvas)
	require.NotNil(t, protoReq.Canvas.Spec)
	require.Len(t, protoReq.Canvas.Spec.Nodes, 6)

	// Check proto nodes after gRPC gateway deserialization
	for i, node := range protoReq.Canvas.Spec.Nodes {
		if node.Type != componentpb.Node_TYPE_WIDGET {
			continue
		}
		if node.Configuration == nil {
			t.Errorf("PROTO: node %d (%s) has nil Configuration after protojson unmarshal", i, node.Id)
		} else {
			config := node.Configuration.AsMap()
			t.Logf("PROTO node %d (%s) configuration: %v", i, node.Id, config)
			if len(config) == 0 {
				t.Errorf("PROTO: node %d (%s) has empty Configuration after protojson unmarshal", i, node.Id)
			}
		}
	}

	// === Step 3: Server-side ProtoToNodes (same as ParseCanvas calls) ===
	modelNodes := ProtoToNodes(protoReq.Canvas.Spec.Nodes)
	require.Len(t, modelNodes, 6)

	for i, node := range modelNodes {
		if node.Type != models.NodeTypeWidget {
			continue
		}
		if node.Configuration == nil {
			t.Errorf("MODEL: node %d (%s) has nil Configuration after ProtoToNodes", i, node.ID)
		} else {
			t.Logf("MODEL node %d (%s) configuration: %v", i, node.ID, node.Configuration)
			if len(node.Configuration) == 0 {
				t.Errorf("MODEL: node %d (%s) has empty Configuration after ProtoToNodes", i, node.ID)
			}
		}
	}

	// === Step 4: DB roundtrip (JSON marshal → unmarshal, same as datatypes.JSONSlice) ===
	dbJSON, err := json.Marshal(modelNodes)
	require.NoError(t, err)

	var dbNodes []models.Node
	err = json.Unmarshal(dbJSON, &dbNodes)
	require.NoError(t, err)
	require.Len(t, dbNodes, 6)

	for i, node := range dbNodes {
		if node.Type != models.NodeTypeWidget {
			continue
		}
		if node.Configuration == nil {
			t.Errorf("DB: node %d (%s) has nil Configuration after DB roundtrip", i, node.ID)
		} else {
			t.Logf("DB node %d (%s) configuration: %v", i, node.ID, node.Configuration)
		}
	}

	// === Step 5: Response serialization (NodesToProto, same as SerializeCanvasVersion) ===
	responseProto := NodesToProto(dbNodes)
	require.Len(t, responseProto, 6)

	for i, node := range responseProto {
		if node.Type != componentpb.Node_TYPE_WIDGET {
			continue
		}
		if node.Configuration == nil {
			t.Errorf("RESPONSE: node %d (%s) has nil Configuration in response proto", i, node.Id)
		} else {
			config := node.Configuration.AsMap()
			t.Logf("RESPONSE node %d (%s) configuration: %v", i, node.Id, config)
			assert.NotEmpty(t, config["label"], "RESPONSE: node %d (%s) missing label", i, node.Id)
			assert.NotEmpty(t, config["color"], "RESPONSE: node %d (%s) missing color", i, node.Id)
			assert.NotNil(t, config["childNodeIds"], "RESPONSE: node %d (%s) missing childNodeIds", i, node.Id)
		}
	}

	// === Step 6: Response JSON (what CLI receives back, simulating protojson marshal) ===
	var responseCanvas canvaspb.Canvas
	responseCanvas.Spec = &canvaspb.Canvas_Spec{
		Nodes: responseProto,
	}
	responseJSON, err := protojson.Marshal(&responseCanvas)
	require.NoError(t, err)
	t.Logf("Response JSON:\n%s", string(responseJSON))

	// Parse the response JSON and verify all groups have configuration
	var responseMap map[string]any
	err = json.Unmarshal(responseJSON, &responseMap)
	require.NoError(t, err)

	respSpec, _ := responseMap["spec"].(map[string]any)
	require.NotNil(t, respSpec)
	respNodes, _ := respSpec["nodes"].([]any)
	require.Len(t, respNodes, 6)

	groupCount := 0
	for i, raw := range respNodes {
		node, _ := raw.(map[string]any)
		nodeType, _ := node["type"].(string)
		if nodeType != "TYPE_WIDGET" {
			continue
		}
		groupCount++
		config, hasConfig := node["configuration"]
		if !hasConfig || config == nil {
			t.Errorf("FINAL JSON: group node %d (%s) has NO configuration — this is the bug!", i, node["id"])
		} else {
			configMap, _ := config.(map[string]any)
			assert.NotEmpty(t, configMap["label"], "FINAL JSON: node %d (%s) missing label", i, node["id"])
			assert.NotEmpty(t, configMap["color"], "FINAL JSON: node %d (%s) missing color", i, node["id"])
		}
	}
	assert.Equal(t, 4, groupCount, "should have found 4 group widgets")

	// === Step 7: Full DescribeCanvasResponse with status (as canvases get returns) ===
	// The get path includes status with executions that also have Struct fields.
	// Test that protojson.Marshal of the full response doesn't corrupt node configs.
	fullResponse := &canvaspb.DescribeCanvasResponse{
		Canvas: &canvaspb.Canvas{
			Metadata: &canvaspb.Canvas_Metadata{
				Id:   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Name: "test-groups",
			},
			Spec: &canvaspb.Canvas_Spec{
				Nodes: responseProto,
			},
		},
	}

	fullJSON, err := protojson.Marshal(fullResponse)
	require.NoError(t, err)

	var fullMap map[string]any
	err = json.Unmarshal(fullJSON, &fullMap)
	require.NoError(t, err)

	canvasMap, _ := fullMap["canvas"].(map[string]any)
	require.NotNil(t, canvasMap)
	fullSpec, _ := canvasMap["spec"].(map[string]any)
	require.NotNil(t, fullSpec)
	fullNodes, _ := fullSpec["nodes"].([]any)
	require.Len(t, fullNodes, 6)

	for i, raw := range fullNodes {
		node, _ := raw.(map[string]any)
		nodeType, _ := node["type"].(string)
		if nodeType != "TYPE_WIDGET" {
			continue
		}
		config, hasConfig := node["configuration"]
		if !hasConfig || config == nil {
			t.Errorf("FULL RESPONSE JSON: group node %d (%s) has NO configuration", i, node["id"])
		}
	}
}
