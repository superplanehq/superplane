import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID } from "../../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../../__fixtures__/lineMetricsFactoriesFixture";

/**
 * Storybook stand-in for the real app after first-run analysis.
 * Opens the Semaphore line board. Production will open the workspace line
 * after analysis finishes.
 */
export function FirstRunBoardExit() {
  return (
    <div data-testid="first-run-board">
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factoriesFixture={lineMetricsFactoriesFixture}
        enableOnboarding={false}
      />
    </div>
  );
}
