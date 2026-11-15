package sso

import (
	"github.com/superplanehq/superplane/ee/billing"
	"github.com/superplanehq/superplane/ee/billing/polar"
	pkgbilling "github.com/superplanehq/superplane/pkg/billing"
)

func init() {
	pkgbilling.Bind(billing.New(polar.NewRuntime()))
}
