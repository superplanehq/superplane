package expressionvalidation

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// stringVisitor receives every string leaf reachable from a configuration map,
// along with the field type derived from the schema (or empty when the value
// lies outside the schema).
type stringVisitor func(fieldPath, fieldType, value string)

// walkConfiguration traverses a configuration map and calls visit for each
// string leaf. Schema-aware where possible: keys present in fields recurse
// using the field's declared structure (object schemas, list item definitions);
// keys outside the schema fall through to a generic recursion.
func walkConfiguration(config map[string]any, fields []configuration.Field, visit stringVisitor) {
	fieldsByName := make(map[string]configuration.Field, len(fields))
	for _, f := range fields {
		fieldsByName[f.Name] = f
	}

	for key, value := range config {
		if field, ok := fieldsByName[key]; ok {
			walkFieldValue(key, field, value, visit)
			continue
		}
		walkUntypedValue(key, value, visit)
	}
}

func walkFieldValue(path string, field configuration.Field, value any, visit stringVisitor) {
	switch field.Type {
	case configuration.FieldTypeObject:
		if field.TypeOptions != nil && field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
			if obj, ok := asAnyMap(value); ok {
				walkConfiguration(obj, field.TypeOptions.Object.Schema, func(p, t, v string) {
					visit(path+"."+p, t, v)
				})
				return
			}
		}
		walkUntypedValue(path, value, visit)

	case configuration.FieldTypeList:
		var itemDef *configuration.ListItemDefinition
		if field.TypeOptions != nil && field.TypeOptions.List != nil {
			itemDef = field.TypeOptions.List.ItemDefinition
		}
		if list, ok := value.([]any); ok {
			walkList(path, list, itemDef, visit)
			return
		}
		walkUntypedValue(path, value, visit)

	default:
		if str, ok := value.(string); ok {
			visit(path, field.Type, str)
			return
		}
		walkUntypedValue(path, value, visit)
	}
}

func walkList(path string, list []any, itemDef *configuration.ListItemDefinition, visit stringVisitor) {
	for i, item := range list {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if itemDef != nil && itemDef.Type == configuration.FieldTypeObject && len(itemDef.Schema) > 0 {
			if obj, ok := asAnyMap(item); ok {
				walkConfiguration(obj, itemDef.Schema, func(p, t, v string) {
					visit(itemPath+"."+p, t, v)
				})
				continue
			}
		}
		if itemDef != nil && itemDef.Type != "" && itemDef.Type != configuration.FieldTypeObject {
			if str, ok := item.(string); ok {
				visit(itemPath, itemDef.Type, str)
				continue
			}
		}
		walkUntypedValue(itemPath, item, visit)
	}
}

func walkUntypedValue(path string, value any, visit stringVisitor) {
	switch v := value.(type) {
	case string:
		visit(path, "", v)
	case map[string]any:
		for key, child := range v {
			walkUntypedValue(path+"."+key, child, visit)
		}
	case map[string]string:
		for key, child := range v {
			visit(path+"."+key, "", child)
		}
	case []any:
		for i, child := range v {
			walkUntypedValue(fmt.Sprintf("%s[%d]", path, i), child, visit)
		}
	}
}

func asAnyMap(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return v, true
	case map[string]string:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[k] = vv
		}
		return out, true
	default:
		return nil, false
	}
}
