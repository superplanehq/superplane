import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import { ACME_ONBOARDING_FACTORY_KEY, ACME_ONBOARDING_LINE_ID } from "../../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../../__fixtures__/lineMetricsFactoriesFixture";

/**
 * Storybook stand-in for the real app after first-run analysis.
 * Opens Acme onboarding Intake with GitHub issues selected.
 * Production will open the workspace line board instead of this fixture.
 */
export function FirstRunBoardExit() {
  return (
    <div data-testid="first-run-board">
      <FactoriesHarness
        pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&source=github-issues`}
        factoriesFixture={lineMetricsFactoriesFixture}
        enableOnboarding={false}
      />
    </div>
  );
}
