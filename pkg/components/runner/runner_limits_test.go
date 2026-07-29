package runner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEnsureRunnerMinutesAvailable(t *testing.T) {
	t.Run("no checker allows the task", func(t *testing.T) {
		setRunnerMinutesLimitCheckerForTest(t, nil)

		require.NoError(t, ensureRunnerMinutesAvailable(core.ExecutionContext{OrganizationID: "org-id"}))
	})

	t.Run("limit message drops the grpc envelope", func(t *testing.T) {
		setRunnerMinutesLimitCheckerForTest(t, func(string) error {
			return status.Error(codes.ResourceExhausted, "organization runner minutes limit exceeded")
		})

		err := ensureRunnerMinutesAvailable(core.ExecutionContext{OrganizationID: "org-id"})
		require.Error(t, err)
		assert.Equal(t, "organization runner minutes limit exceeded", err.Error())
	})

	t.Run("plain errors pass through unchanged", func(t *testing.T) {
		setRunnerMinutesLimitCheckerForTest(t, func(string) error {
			return errors.New("usage service unavailable")
		})

		err := ensureRunnerMinutesAvailable(core.ExecutionContext{OrganizationID: "org-id"})
		require.Error(t, err)
		assert.Equal(t, "usage service unavailable", err.Error())
	})
}

func setRunnerMinutesLimitCheckerForTest(t *testing.T, checker RunnerMinutesLimitChecker) {
	t.Helper()

	previous := runnerMinutesLimitChecker
	runnerMinutesLimitChecker = checker
	t.Cleanup(func() { runnerMinutesLimitChecker = previous })
}
