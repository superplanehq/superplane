package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Decoder struct {
	raw []byte
}

func NewDecoder(raw []byte) *Decoder {
	return &Decoder{raw: raw}
}

func (d *Decoder) DecodeYAML(out any) error {
	// We intentionally decode YAML by round-tripping through JSON:
	// 1. Parse YAML with yaml.v3 to preserve YAML behavior used by existing CLI files.
	// 2. Marshal to JSON.
	// 3. Decode with json.Decoder + DisallowUnknownFields so unknown keys fail fast.
	//
	// This is deliberate because our CLI resource structs and generated API models
	// are keyed by json tags (camelCase API field names). Using yaml.v3 KnownFields
	// directly would validate against YAML field names/tags, which is not aligned
	// with those json-tagged types.
	var yamlObject any
	if err := yaml.Unmarshal(d.raw, &yamlObject); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	jsonData, err := json.Marshal(yamlObject)
	if err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	return nil
}
