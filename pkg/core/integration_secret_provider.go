package core

import (
	"github.com/sirupsen/logrus"
)

/*
 * IntegrationSecretContext is the context given to integrations when resolving exportable secrets.
 */
type IntegrationSecretContext struct {
	Logger      *logrus.Entry
	HTTP        HTTPContext
	Integration IntegrationContext
}

/*
 * IntegrationSecrets is the materialization of an integration's exportable
 * credentials, plus optional agent-facing guidance and setup.
 */
type IntegrationSecrets struct {
	Values    map[string][]byte
	Usage     string
	Setup     string
	SetupName string
}

/*
 * IntegrationSecretProvider is an optional integration capability for materializing
 * key/value secrets that other parts of the system may consume (runners, components, etc.).
 */
type IntegrationSecretProvider interface {
	ResolveSecrets(ctx IntegrationSecretContext) (IntegrationSecrets, error)
}
