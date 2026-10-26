package auth_test

import (
	eebilling "github.com/superplanehq/superplane/ee/billing"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing"
)

func init() { billing.Bind(eebilling.New(polar.NewRuntime())) }
