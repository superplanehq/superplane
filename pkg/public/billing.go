package public

import "github.com/superplanehq/superplane/pkg/billing"

// SetBillingHTTPRegistrar installs billing routes before the web routes start.
func (s *Server) SetBillingHTTPRegistrar(registrar billing.HTTPRegistrar) {
	s.billingHTTPRegistrar = registrar
	registrar.RegisterPublic(s.publicRouter)
	registrar.RegisterAdmin(s.adminRouter)
}
