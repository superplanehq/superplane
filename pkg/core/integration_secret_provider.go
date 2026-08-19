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
 * IntegrationSecretProvider is an optional integration capability for materializing
 * key/value secrets that other parts of the system may consume (runners, components, etc.).
 */
type IntegrationSecretProvider interface {
	ResolveSecrets(ctx IntegrationSecretContext) (map[string][]byte, error)
}
