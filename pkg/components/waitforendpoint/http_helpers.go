package waitforendpoint

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	AuthorizationTypeBearer       = "bearer"
	AuthorizationTypeBasicAuth    = "basic_auth"
	AuthorizationTypeCustomHeader = "custom_header"
)

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuthorizationSpec struct {
	Type       string                      `json:"type" mapstructure:"type"`
	Credential *configuration.SecretKeyRef `json:"credential,omitempty" mapstructure:"credential"`
	Username   *string                     `json:"username,omitempty" mapstructure:"username"`
	Password   *configuration.SecretKeyRef `json:"password,omitempty" mapstructure:"password"`
	HeaderName *string                     `json:"headerName,omitempty" mapstructure:"headerName"`
	Value      *configuration.SecretKeyRef `json:"value,omitempty" mapstructure:"value"`
}

func AuthorizationField() configuration.Field {
	return configuration.Field{
		Name:        "authorization",
		Label:       "Authorization",
		Type:        configuration.FieldTypeObject,
		Required:    false,
		Togglable:   true,
		Description: "Configure request authentication",
		TypeOptions: &configuration.TypeOptions{
			Object: &configuration.ObjectTypeOptions{
				Schema: []configuration.Field{
					{
						Name:        "type",
						Label:       "Type",
						Type:        configuration.FieldTypeSelect,
						Required:    true,
						Default:     AuthorizationTypeBearer,
						Description: "Authorization method to use for this request",
						TypeOptions: &configuration.TypeOptions{
							Select: &configuration.SelectTypeOptions{
								Options: []configuration.FieldOption{
									{Label: "Bearer Token", Value: AuthorizationTypeBearer},
									{Label: "Basic Auth", Value: AuthorizationTypeBasicAuth},
									{Label: "Custom Header", Value: AuthorizationTypeCustomHeader},
								},
							},
						},
					},
					{
						Name:        "credential",
						Label:       "Token",
						Type:        configuration.FieldTypeSecretKey,
						Description: "Secret and key that stores the bearer token",
						VisibilityConditions: []configuration.VisibilityCondition{
							{Field: "type", Values: []string{AuthorizationTypeBearer}},
						},
						RequiredConditions: []configuration.RequiredCondition{
							{Field: "type", Values: []string{AuthorizationTypeBearer}},
						},
					},
					{
						Name:        "username",
						Label:       "Username",
						Type:        configuration.FieldTypeString,
						Description: "Username for Basic Auth",
						VisibilityConditions: []configuration.VisibilityCondition{
							{Field: "type", Values: []string{AuthorizationTypeBasicAuth}},
						},
						RequiredConditions: []configuration.RequiredCondition{
							{Field: "type", Values: []string{AuthorizationTypeBasicAuth}},
						},
					},
					{
						Name:        "password",
						Label:       "Password",
						Type:        configuration.FieldTypeSecretKey,
						Description: "Secret and key that stores the Basic Auth password",
						VisibilityConditions: []configuration.VisibilityCondition{
							{Field: "type", Values: []string{AuthorizationTypeBasicAuth}},
						},
						RequiredConditions: []configuration.RequiredCondition{
							{Field: "type", Values: []string{AuthorizationTypeBasicAuth}},
						},
					},
					{
						Name:        "headerName",
						Label:       "Header name",
						Type:        configuration.FieldTypeString,
						Description: "Custom header name that will receive the secret value",
						Placeholder: "X-API-Key",
						VisibilityConditions: []configuration.VisibilityCondition{
							{Field: "type", Values: []string{AuthorizationTypeCustomHeader}},
						},
						RequiredConditions: []configuration.RequiredCondition{
							{Field: "type", Values: []string{AuthorizationTypeCustomHeader}},
						},
					},
					{
						Name:        "value",
						Label:       "Value",
						Type:        configuration.FieldTypeSecretKey,
						Description: "Secret and key that stores the custom header value",
						VisibilityConditions: []configuration.VisibilityCondition{
							{Field: "type", Values: []string{AuthorizationTypeCustomHeader}},
						},
						RequiredConditions: []configuration.RequiredCondition{
							{Field: "type", Values: []string{AuthorizationTypeCustomHeader}},
						},
					},
				},
			},
		},
	}
}

func ValidateAuthorization(spec *AuthorizationSpec) error {
	if spec == nil {
		return nil
	}

	switch spec.Type {
	case "":
		return fmt.Errorf("authorization: type is required when authorization is set")
	case AuthorizationTypeBearer:
		return validateSecretKeyRef("authorization bearer credential", spec.Credential)
	case AuthorizationTypeBasicAuth:
		if spec.Username == nil || *spec.Username == "" {
			return fmt.Errorf("authorization basic auth: username is required")
		}
		return validateSecretKeyRef("authorization basic auth password", spec.Password)
	case AuthorizationTypeCustomHeader:
		if spec.HeaderName == nil || *spec.HeaderName == "" {
			return fmt.Errorf("authorization custom header: header name is required")
		}
		return validateSecretKeyRef("authorization custom header value", spec.Value)
	default:
		return fmt.Errorf("authorization: invalid type: %s", spec.Type)
	}
}

func ApplyAuthorization(secrets core.SecretsContext, spec *AuthorizationSpec, request *http.Request) (string, error) {
	if spec == nil {
		return "", nil
	}
	if err := ValidateAuthorization(spec); err != nil {
		return "", err
	}
	if secrets == nil {
		return "", fmt.Errorf("authorization: secrets context is not available")
	}

	switch spec.Type {
	case AuthorizationTypeBearer:
		value, err := resolveAuthorizationSecret(secrets, "bearer credential", spec.Credential)
		if err != nil {
			return "", err
		}
		request.Header.Set("Authorization", "Bearer "+string(value))
		return "Authorization", nil
	case AuthorizationTypeBasicAuth:
		password, err := resolveAuthorizationSecret(secrets, "basic auth password", spec.Password)
		if err != nil {
			return "", err
		}
		request.SetBasicAuth(*spec.Username, string(password))
		return "Authorization", nil
	case AuthorizationTypeCustomHeader:
		value, err := resolveAuthorizationSecret(secrets, "custom header value", spec.Value)
		if err != nil {
			return "", err
		}
		request.Header.Set(*spec.HeaderName, string(value))
		return *spec.HeaderName, nil
	default:
		return "", fmt.Errorf("authorization: invalid type: %s", spec.Type)
	}
}

func ValidateStatusMatcher(matcher string) error {
	if strings.TrimSpace(matcher) == "" {
		return fmt.Errorf("expected status is required")
	}

	for _, value := range strings.Split(matcher, ",") {
		value = strings.TrimSpace(value)
		if len(value) == 3 && strings.HasSuffix(value, "xx") {
			prefix, err := strconv.Atoi(value[:1])
			if err == nil && prefix >= 1 && prefix <= 5 {
				continue
			}
		}

		code, err := strconv.Atoi(value)
		if err != nil || code < 100 || code > 599 {
			return fmt.Errorf("invalid HTTP status matcher: %s", value)
		}
	}

	return nil
}

func MatchesStatus(statusCode int, matcher string) bool {
	for _, value := range strings.Split(matcher, ",") {
		value = strings.TrimSpace(value)
		if len(value) == 3 && strings.HasSuffix(value, "xx") {
			status := strconv.Itoa(statusCode)
			if strings.HasPrefix(status, value[:1]) {
				return true
			}
			continue
		}

		expectedCode, err := strconv.Atoi(value)
		if err == nil && statusCode == expectedCode {
			return true
		}
	}

	return false
}

func validateSecretKeyRef(name string, ref *configuration.SecretKeyRef) error {
	if ref != nil && ref.IsSet() {
		return nil
	}
	if ref != nil && (ref.Secret != "" || ref.Key != "") {
		return fmt.Errorf("%s: both organization secret and key name are required", name)
	}
	return fmt.Errorf("%s: organization secret and key are required", name)
}

func resolveAuthorizationSecret(
	secrets core.SecretsContext,
	name string,
	ref *configuration.SecretKeyRef,
) ([]byte, error) {
	if ref == nil {
		return nil, fmt.Errorf("authorization %s: organization secret and key are required", name)
	}
	value, err := secrets.GetKey(ref.Secret, ref.Key)
	if err != nil {
		if errors.Is(err, core.ErrSecretKeyNotFound) {
			return nil, fmt.Errorf("authorization %s: %w", name, err)
		}
		return nil, fmt.Errorf("authorization %s: resolve secret: %w", name, err)
	}

	return value, nil
}
