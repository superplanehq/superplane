package dockerhub

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type RepositoryMetadata struct {
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Name      string `json:"name" mapstructure:"name"`
	URL       string `json:"url" mapstructure:"url"`
}

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
