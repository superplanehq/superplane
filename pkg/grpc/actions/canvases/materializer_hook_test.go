package canvases

import (
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
)

// Canvas write handlers are worker-authoritative: they register a pending version
// row and hand off materialization to the RepositoryMaterializerWorker, which is
// not running in unit tests. This wires support.Setup to run that materialization
// synchronously in-process so handler tests observe materialized state.
//
// It lives in the canvases test package (which already imports both support and
// materialize) rather than in support itself, because support importing
// materialize would create a test import cycle.
func init() {
	support.InProcessMaterializerHook = func(
		gitProvider git.Provider,
		reg *registry.Registry,
		encryptor crypto.Encryptor,
		authService authorization.Authorization,
		webhookBaseURL string,
	) {
		materialize.SetInProcessMaterializer(&materialize.BranchMaterializer{
			GitProvider:    gitProvider,
			Registry:       reg,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
		})
	}
}
