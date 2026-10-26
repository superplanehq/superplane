package public

import (
	eebilling "github.com/superplanehq/superplane/ee/billing"
	billinghttp "github.com/superplanehq/superplane/ee/billing/http"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/test/support"
)

func init() {
	billing.Bind(eebilling.New(polar.NewRuntime()))
}

func installBillingHTTPTestRoutes(server *Server, resources *support.ResourceRegistry) {
	server.SetBillingHTTPRegistrar(billinghttp.New(resources.Encryptor, resources.Registry, func() (string, string) {
		return server.BaseURL, server.WebhooksBaseURL
	}))
}
