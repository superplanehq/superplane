package components

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/components"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
)

func ConfigurationFieldToProto(field components.ConfigurationField) *pb.ConfigurationField {
	pbField := &pb.ConfigurationField{
		Name:        field.Name,
		Label:       field.Label,
		Type:        field.Type,
		Description: field.Description,
		Required:    field.Required,
	}

	// Handle default value
	if field.Default != nil {
		// Convert default value to JSON string for proto
		defaultBytes, err := json.Marshal(field.Default)
		if err == nil {
			defaultStr := string(defaultBytes)
			pbField.DefaultValue = &defaultStr
		}
	}

	// Handle options (for select/multi_select)
	if len(field.Options) > 0 {
		pbField.Options = make([]*pb.FieldOption, len(field.Options))
		for i, opt := range field.Options {
			pbField.Options[i] = &pb.FieldOption{
				Label: opt.Label,
				Value: opt.Value,
			}
		}
	}

	// Handle min/max (for number type)
	if field.Min != nil {
		min := int32(*field.Min)
		pbField.Min = &min
	}
	if field.Max != nil {
		max := int32(*field.Max)
		pbField.Max = &max
	}

	// Handle list item definition (for list type)
	if field.ListItem != nil {
		pbField.ListItem = &pb.ListItemDefinition{
			Type: field.ListItem.Type,
		}
		if len(field.ListItem.Schema) > 0 {
			pbField.ListItem.Schema = make([]*pb.ConfigurationField, len(field.ListItem.Schema))
			for i, schemaField := range field.ListItem.Schema {
				pbField.ListItem.Schema[i] = ConfigurationFieldToProto(schemaField)
			}
		}
	}

	// Handle object schema (for object type)
	if len(field.Schema) > 0 {
		pbField.Schema = make([]*pb.ConfigurationField, len(field.Schema))
		for i, schemaField := range field.Schema {
			pbField.Schema[i] = ConfigurationFieldToProto(schemaField)
		}
	}

	return pbField
}
