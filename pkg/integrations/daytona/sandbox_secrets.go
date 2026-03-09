package daytona

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	SandboxSecretTypeFile   = "file"
	SandboxSecretTypeEnvVar = "env-var"

	sandboxSecretsEnvVarFile = SandboxBaseDir + "/secrets.env.sh"
)

type SandboxSecret struct {
	Type  string                     `json:"type" mapstructure:"type"`
	Path  string                     `json:"path,omitempty" mapstructure:"path"`
	Name  string                     `json:"name,omitempty" mapstructure:"name"`
	Value configuration.SecretKeyRef `json:"value" mapstructure:"value"`
}

type sandboxEnvBinding struct {
	Name  string
	Value []byte
}

func sandboxSecretsConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "secrets",
		Label:       "Secrets",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: "Inject organization secret keys as files or environment variables after the sandbox starts",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Secret",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:     "type",
							Label:    "Type",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							Default:  SandboxSecretTypeFile,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "File", Value: SandboxSecretTypeFile},
										{Label: "Environment Variable", Value: SandboxSecretTypeEnvVar},
									},
								},
							},
						},
						{
							Name:                 "path",
							Label:                "Path",
							Type:                 configuration.FieldTypeString,
							Required:             false,
							Placeholder:          "/home/daytona/.ssh/id_rsa",
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "type", Values: []string{SandboxSecretTypeFile}}},
							RequiredConditions:   []configuration.RequiredCondition{{Field: "type", Values: []string{SandboxSecretTypeFile}}},
						},
						{
							Name:                 "name",
							Label:                "Name",
							Type:                 configuration.FieldTypeString,
							Required:             false,
							Placeholder:          "GITHUB_TOKEN",
							VisibilityConditions: []configuration.VisibilityCondition{{Field: "type", Values: []string{SandboxSecretTypeEnvVar}}},
							RequiredConditions:   []configuration.RequiredCondition{{Field: "type", Values: []string{SandboxSecretTypeEnvVar}}},
						},
						{
							Name:        "value",
							Label:       "Value",
							Type:        configuration.FieldTypeSecretKey,
							Required:    true,
							Description: "Secret and key to inject",
						},
					},
				},
			},
		},
	}
}

func validateSandboxSecrets(secrets []SandboxSecret) error {
	for i, secret := range secrets {
		secretType := strings.TrimSpace(secret.Type)
		if secretType == "" {
			return fmt.Errorf("secrets[%d].type is required", i)
		}

		if !secret.Value.IsSet() {
			return fmt.Errorf("secrets[%d].value.secret and secrets[%d].value.key are required", i, i)
		}

		switch secretType {
		case SandboxSecretTypeFile:
			if strings.TrimSpace(secret.Path) == "" {
				return fmt.Errorf("secrets[%d].path is required for file secrets", i)
			}

		case SandboxSecretTypeEnvVar:
			name := strings.TrimSpace(secret.Name)
			if name == "" {
				return fmt.Errorf("secrets[%d].name is required for env-var secrets", i)
			}

			if !envVariableNamePattern.MatchString(name) {
				return fmt.Errorf("invalid env variable name: %s", secret.Name)
			}

		default:
			return fmt.Errorf("invalid secret type: %s", secret.Type)
		}
	}

	return nil
}

func injectSandboxSecrets(client *Client, sandboxID string, secretsContext core.SecretsContext, secrets []SandboxSecret) error {
	if len(secrets) == 0 {
		return nil
	}

	var envBindings []sandboxEnvBinding
	permissionTargets := make([]string, 0, len(secrets))

	for _, secret := range secrets {
		value, err := secretsContext.GetKey(secret.Value.Secret, secret.Value.Key)
		if err != nil {
			return fmt.Errorf("failed to resolve secret %s/%s: %w", secret.Value.Secret, secret.Value.Key, err)
		}

		switch strings.TrimSpace(secret.Type) {
		case SandboxSecretTypeFile:
			filePath := strings.TrimSpace(secret.Path)
			parentDir := path.Dir(filePath)
			if parentDir != "." && parentDir != "/" {
				if err := ensureFolderExists(client, sandboxID, parentDir); err != nil {
					return err
				}
			}

			if err := client.UploadFile(sandboxID, filePath, value); err != nil {
				return fmt.Errorf("failed to upload secret file %s: %w", filePath, err)
			}

			permissionTargets = append(permissionTargets, filePath)

		case SandboxSecretTypeEnvVar:
			name := strings.TrimSpace(secret.Name)
			envBindings = append(envBindings, sandboxEnvBinding{Name: name, Value: value})
		}
	}

	if len(envBindings) > 0 {
		if err := ensureFolderExists(client, sandboxID, path.Dir(sandboxSecretsEnvVarFile)); err != nil {
			return err
		}

		if err := client.UploadFile(sandboxID, sandboxSecretsEnvVarFile, []byte(buildSandboxSecretsEnvScript(envBindings))); err != nil {
			return fmt.Errorf("failed to upload secrets env script: %w", err)
		}

		permissionTargets = append(permissionTargets, sandboxSecretsEnvVarFile)
	}

	return setSecretFilePermissions(client, sandboxID, permissionTargets)
}

func ensureFolderExists(client *Client, sandboxID, directory string) error {
	err := client.CreateFolder(sandboxID, directory)
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "exist") {
		return nil
	}

	return fmt.Errorf("failed to create folder %s: %w", directory, err)
}

func buildSandboxSecretsEnvScript(bindings []sandboxEnvBinding) string {
	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].Name < bindings[j].Name
	})

	lines := make([]string, 0, len(bindings)+2)
	lines = append(lines, "#!/bin/sh", "# Generated by SuperPlane. Do not edit.")

	for _, binding := range bindings {
		lines = append(lines, fmt.Sprintf("export %s=%s", binding.Name, shellQuote(string(binding.Value))))
	}

	return strings.Join(lines, "\n") + "\n"
}

func setSecretFilePermissions(client *Client, sandboxID string, filePaths []string) error {
	uniqueFiles := uniqueQuotedPaths(filePaths)
	if len(uniqueFiles) == 0 {
		return nil
	}

	command := fmt.Sprintf("chmod 600 %s", strings.Join(uniqueFiles, " "))

	response, err := client.ExecuteCommand(sandboxID, &ExecuteCommandRequest{
		Command: command,
	})
	if err != nil {
		return fmt.Errorf("failed to set secret file permissions: %w", err)
	}

	if response.ExitCode != 0 {
		return fmt.Errorf("failed to set secret file permissions: %s", response.ShortResult())
	}

	return nil
}

func uniqueQuotedPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))

	for _, value := range paths {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, shellQuote(trimmed))
	}

	sort.Strings(result)
	return result
}

func wrapCommandWithSandboxSecretEnv(command string) string {
	return fmt.Sprintf(
		"if [ -f %s ]; then . %s; fi && %s",
		shellQuote(sandboxSecretsEnvVarFile),
		shellQuote(sandboxSecretsEnvVarFile),
		command,
	)
}
