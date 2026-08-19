package runner

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/core"
	"google.golang.org/grpc/status"
)

// RunnerMinutesLimitChecker blocks starting a new runner task when an org is over budget.
// Wired from process startup (see pkg/usage) so this package does not import usage directly.
type RunnerMinutesLimitChecker func(organizationID string) error

var runnerMinutesLimitChecker RunnerMinutesLimitChecker

func SetRunnerMinutesLimitChecker(checker RunnerMinutesLimitChecker) {
	runnerMinutesLimitChecker = checker
}

func ensureRunnerMinutesAvailable(ctx core.ExecutionContext) error {
	if runnerMinutesLimitChecker == nil {
		return nil
	}

	err := runnerMinutesLimitChecker(ctx.OrganizationID)
	if err == nil {
		return nil
	}

	// The checker reports limits as gRPC status errors, but this error becomes the
	// canvas failure reason verbatim, so keep only the human-readable message.
	return errors.New(status.Convert(err).Message())
}
