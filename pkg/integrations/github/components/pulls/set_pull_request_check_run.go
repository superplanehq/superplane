package pulls

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/core"
)

// SetPullRequestCheckRun uses the ordered status lifecycle with Checks API delivery.
type SetPullRequestCheckRun struct{ SetPullRequestCheck }

func (*SetPullRequestCheckRun) Name() string  { return "github.setPullRequestCheckRun" }
func (*SetPullRequestCheckRun) Label() string { return "Set Pull Request Check Run" }
func (*SetPullRequestCheckRun) Description() string {
	return "Publish a running check for a pull request assessment"
}
func (*SetPullRequestCheckRun) Documentation() string {
	return `Publish a GitHub check run for an assessed pull request commit. Use the same configuration as Set Pull Request Check.

Pending shows a running check. Success completes the check successfully. Failure and error complete the check as failed. The output includes the reason and result URL.

The same attempt updates one check run. A stable external ID recovers interrupted creation. New attempts retire running checks, including retries on the same commit. Results stay on the assessed commit.

Cancellation, removal, a closed or changed pull request, and completion without a final result fail the running check. Delivery uses the existing durable retries and ordering.

The GitHub App needs Checks write and Pull requests read permissions. Existing installations must accept the permission update. This component does not change branch rules.`
}
func (*SetPullRequestCheckRun) Execute(ctx core.ExecutionContext) error {
	config, err := decodePullRequestCheck(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Phase != "" || !slices.Contains([]string{"pending", "success", "failure", "error"}, config.State) {
		return fmt.Errorf("state must be pending, success, failure, or error")
	}
	config.CheckRun = true
	return executePullRequestCheck(ctx, config)
}
