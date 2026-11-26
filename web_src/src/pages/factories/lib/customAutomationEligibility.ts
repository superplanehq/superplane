import type { ManualCustomAutomation } from "./customAutomation";
import type { SplitRunFixture } from "../pages/work-order-split-run/splitRunMocks";
import { isClosedWorkOrderDisplayStatus } from "../pages/work-order-split-run/splitRunFooter";

export function eligibleColumnAutomations(
  automations: readonly ManualCustomAutomation[],
  columnId: string,
  fixture: SplitRunFixture,
) {
  return automations.filter((automation) => {
    const savedColumn = automation.columnKey ?? "verify";
    if (savedColumn !== columnId) return false;
    if (savedColumn.startsWith("custom-")) {
      if (
        fixture.currentStageKey !== savedColumn ||
        !fixture.manualStageAppIds?.includes(automation.id) ||
        fixture.queuedStageAppId === automation.id
      )
        return false;
      const latest = [...fixture.phases]
        .reverse()
        .find((phase) => !phase.historyRun && phase.appId === automation.id && phase.columnKey === savedColumn);
      return !latest || latest.status === "failed" || latest.status === "cancelled";
    }
    if (savedColumn === "backlog") return fixture.footer.kind === "draft";
    if (savedColumn === "implement") {
      return (
        fixture.currentStepIndex === 0 &&
        !fixture.canRunVerifyCustomAutomation &&
        (fixture.lineStatus === "running" || fixture.lineStatus === "waiting")
      );
    }
    if (savedColumn === "done") return isClosedWorkOrderDisplayStatus(fixture.footer.status);
    return Boolean(fixture.canRunVerifyCustomAutomation);
  });
}
