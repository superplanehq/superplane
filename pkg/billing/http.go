package billing

import "github.com/gorilla/mux"

// HTTPRegistrar installs billing handlers on the existing routers. The caller
// owns authentication, authorization, and middleware on each router.
type HTTPRegistrar interface {
	RegisterPublic(*mux.Router)
	RegisterAdmin(*mux.Router)
}

// InstallationSettings contains the billing fields on shared installation
// settings. Provider configuration belongs to core.
type InstallationSettings struct {
	TrialCreditCents            int64
	MarkupBPS                   int
	WarningThresholdBPS         int
	CreditPurchaseFeeBPS        int
	CreditPurchaseFixedFeeCents int64
}
