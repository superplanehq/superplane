package configuration

import (
	"fmt"
	"strings"
)

/*
 * SecretRef identifies an organization secret by name.
 */
type SecretRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
}

func (r SecretRef) IsSet() bool {
	return strings.TrimSpace(r.Secret) != ""
}

/*
 * SecretKeyRef identifies a secret key by name.
 */
type SecretKeyRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

func (r SecretKeyRef) IsSet() bool {
	return r.Secret != "" && r.Key != ""
}

/*
 * IntegrationRef identifies a connected integration instance by installation name.
 */
type IntegrationRef struct {
	Name string `json:"name" mapstructure:"name"`
}

func (r IntegrationRef) IsSet() bool {
	return strings.TrimSpace(r.Name) != ""
}

func DecodeSecretRef(value any) (SecretRef, error) {
	switch typed := value.(type) {
	case map[string]any:
		return decodeSecretRefObject(typed)

	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, val := range typed {
			keyString, ok := key.(string)
			if !ok {
				continue
			}
			normalized[keyString] = val
		}
		return decodeSecretRefObject(normalized)

	default:
		if value == nil {
			return SecretRef{}, nil
		}
		return SecretRef{}, fmt.Errorf("must be an object")
	}
}

func decodeSecretRefObject(value map[string]any) (SecretRef, error) {
	ref := SecretRef{}
	if secret, ok := value["secret"].(string); ok {
		ref.Secret = strings.TrimSpace(secret)
	}
	return ref, nil
}

// DecodeIntegrationRef accepts a {name} object.
func DecodeIntegrationRef(value any) (IntegrationRef, error) {
	switch typed := value.(type) {
	case map[string]any:
		return decodeIntegrationRefObject(typed)

	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, val := range typed {
			keyString, ok := key.(string)
			if !ok {
				continue
			}
			normalized[keyString] = val
		}
		return decodeIntegrationRefObject(normalized)

	default:
		if value == nil {
			return IntegrationRef{}, nil
		}
		return IntegrationRef{}, fmt.Errorf("must be an object")
	}
}

func decodeIntegrationRefObject(value map[string]any) (IntegrationRef, error) {
	ref := IntegrationRef{}

	if name, ok := value["name"].(string); ok {
		ref.Name = strings.TrimSpace(name)
	}

	return ref, nil
}
