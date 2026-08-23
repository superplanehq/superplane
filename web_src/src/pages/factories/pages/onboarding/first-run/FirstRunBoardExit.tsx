import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../../__fixtures__/lineMetricsFactoriesFixture";

/**
 * Storybook stand-in for the real app after first-run analysis.
 * Opens Intake with GitHub issues selected so tickets in analysis are visible.
 * Production will open the workspace line board instead of this fixture.
 */
export function FirstRunBoardExit() {
  const line = REFUND_FACTORY_LINES[0];

  return (
    <div data-testid="first-run-board">
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&source=github-issues`}
        factoriesFixture={lineMetricsFactoriesFixture}
        enableOnboarding={false}
      />
    </div>
  );
}
