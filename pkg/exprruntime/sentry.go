package exprruntime

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
)

// SentryIssueDescriptionOption shares Sentry description assembly with imports.
func SentryIssueDescriptionOption() expr.Option {
	return expr.Function("sentryIssueDescription", func(params ...any) (any, error) {
		if len(params) != 4 {
			return nil, fmt.Errorf("sentryIssueDescription() expects four arguments")
		}
		return sentry.AssembleIssueDescription(params[0], params[1], params[2], params[3]), nil
	}, new(func(any, any, any, any) string))
}

// SentryEventHasStackOption shares Sentry event decoding and stack detection.
func SentryEventHasStackOption() expr.Option {
	return expr.Function("sentryEventHasStack", func(params ...any) (any, error) {
		if len(params) != 1 {
			return nil, fmt.Errorf("sentryEventHasStack() expects one argument")
		}
		return sentry.EventHasStack(params[0]), nil
	}, new(func(any) bool))
}
