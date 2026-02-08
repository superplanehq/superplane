package dockerhub

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

func resolveNamespace(configNamespace string, integration core.IntegrationContext) (string, error) {
	namespace := strings.TrimSpace(configNamespace)
	if namespace != "" {
		return namespace, nil
	}

	if integration == nil {
		return "", fmt.Errorf("namespace is required")
	}

	username, err := integration.GetConfig("username")
	if err != nil {
		return "", fmt.Errorf("username is required")
	}

	namespace = strings.TrimSpace(string(username))
	if namespace == "" {
		return "", fmt.Errorf("username is required")
	}

	return namespace, nil
}

func repositoryNameFromEvent(repoName, name string) string {
	if name != "" {
		return strings.TrimSpace(name)
	}

	parts := strings.Split(strings.TrimSpace(repoName), "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func findSecret(ctx core.IntegrationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}
