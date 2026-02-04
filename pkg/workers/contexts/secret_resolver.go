package contexts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// splitSecretAndKey splits "secretId:keyName" on the first colon. Key names may contain colons.
func splitSecretAndKey(ref string) (secretID, keyName string) {
	i := strings.Index(ref, ":")
	if i < 0 {
		return ref, ""
	}
	return ref[:i], ref[i+1:]
}

func secretValueByKey(data any, key string) string {
	m, ok := data.(map[string]any)
	if !ok || key == "" {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// NewSecretResolver returns a SecretResolver that looks up secrets by reference in the given
// transaction and domain, decrypts the secret data, and returns it as a map.
// The resolved value is the secret's local data (map of key-value pairs).
func NewSecretResolver(tx *gorm.DB, domainType string, domainID uuid.UUID, encryptor crypto.Encryptor) SecretResolver {
	return func(secretID string) (any, error) {
		secret, err := models.FindSecretByIDInTransaction(tx, domainType, domainID, secretID)
		if err != nil {
			return nil, err
		}

		decrypted, err := encryptor.Decrypt(context.Background(), secret.Data, []byte(secret.Name))
		if err != nil {
			return nil, err
		}

		var data map[string]string
		if err := json.Unmarshal(decrypted, &data); err != nil {
			return nil, err
		}

		// Return as map[string]any for consistency with JSON config
		result := make(map[string]any, len(data))
		for k, v := range data {
			result[k] = v
		}
		return result, nil
	}
}

// ResolveSecretReferencesInConfig resolves only secret field references in config to their
// decrypted values. All other fields are copied as-is. Used at execution time so that
// persisted config stores only secret references; resolution happens in memory when running the component.
func ResolveSecretReferencesInConfig(config map[string]any, fields []configuration.Field, resolver SecretResolver) (map[string]any, error) {
	if resolver == nil {
		return config, nil
	}
	fieldsByName := make(map[string]configuration.Field, len(fields))
	for _, f := range fields {
		fieldsByName[f.Name] = f
	}
	return resolveSecretReferencesWithSchema(config, fieldsByName, resolver)
}

func resolveSecretReferencesWithSchema(config map[string]any, fieldsByName map[string]configuration.Field, resolver SecretResolver) (map[string]any, error) {
	result := make(map[string]any, len(config))
	for key, value := range config {
		field, ok := fieldsByName[key]
		if !ok {
			result[key] = value
			continue
		}
		resolved, err := resolveSecretReferenceValue(value, field, fieldsByName, resolver)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", key, err)
		}
		result[key] = resolved
	}
	return result, nil
}

func resolveSecretReferenceValue(value any, field configuration.Field, fieldsByName map[string]configuration.Field, resolver SecretResolver) (any, error) {
	if field.Type == configuration.FieldTypeSecret {
		if ref, ok := value.(string); ok && ref != "" {
			return resolver(ref)
		}
		return value, nil
	}
	if field.Type == configuration.FieldTypeSecretAndKey {
		if ref, ok := value.(string); ok && ref != "" {
			secretID, keyName := splitSecretAndKey(ref)
			if secretID == "" || keyName == "" {
				return value, nil
			}
			data, err := resolver(secretID)
			if err != nil {
				return nil, err
			}
			return secretValueByKey(data, keyName), nil
		}
		return value, nil
	}
	if field.TypeOptions != nil && field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
		if obj, ok := asAnyMapForSecret(value); ok {
			nestedByName := make(map[string]configuration.Field)
			for _, f := range field.TypeOptions.Object.Schema {
				nestedByName[f.Name] = f
			}
			return resolveSecretReferencesWithSchema(obj, nestedByName, resolver)
		}
	}
	if field.TypeOptions != nil && field.TypeOptions.List != nil && field.TypeOptions.List.ItemDefinition != nil {
		itemDef := field.TypeOptions.List.ItemDefinition
		if list, ok := value.([]any); ok {
			out := make([]any, len(list))
			for i, item := range list {
				if itemDef.Type == configuration.FieldTypeObject && len(itemDef.Schema) > 0 {
					if itemMap, ok := asAnyMapForSecret(item); ok {
						nestedByName := make(map[string]configuration.Field)
						for _, f := range itemDef.Schema {
							nestedByName[f.Name] = f
						}
						resolved, err := resolveSecretReferencesWithSchema(itemMap, nestedByName, resolver)
						if err != nil {
							return nil, fmt.Errorf("list item %d: %w", i, err)
						}
						out[i] = resolved
						continue
					}
				}
				out[i] = item
			}
			return out, nil
		}
	}
	return value, nil
}

func asAnyMapForSecret(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[string]string:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out, true
	default:
		return nil, false
	}
}
