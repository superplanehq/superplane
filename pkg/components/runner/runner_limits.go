package runner

import (
	"github.com/superplanehq/superplane/pkg/core"
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
	return runnerMinutesLimitChecker(ctx.OrganizationID)
}
