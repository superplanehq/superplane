import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import type { CreateWithAgentView } from "../createWithAgentTypes";
import { PLANNING_SESSION_PHASE_ID, planningSessionPhase } from "../planningSessionActivity";
import { headerSpendFromUsageSeries } from "./planningHeaderSpend";
import { useReportLiveHeaderSpend } from "./liveHeaderSpendContext";

type PlanningSpendView = Pick<CreateWithAgentView, "canvasId" | "executionId" | "messages" | "machineStatus">;

function terminalCommandStatus(status: string | undefined): "passed" | "failed" | null {
  if (status === "failed") {
    return "failed";
  }
  if (status === "passed") {
    return "passed";
  }
  return null;
}

export function PlanningHeaderSpendCollector({
  organizationId,
  view,
}: {
  organizationId?: string;
  view: PlanningSpendView;
}) {
  const line = planningSessionPhase(view).stream[0];
  const { usageSeries } = useLiveLogStream(
    line?.executionId ?? "",
    line?.status === "running",
    terminalCommandStatus(line?.status),
    null,
    { organizationId, canvasId: view.canvasId },
  );
  const spend = headerSpendFromUsageSeries(usageSeries);
  useReportLiveHeaderSpend(PLANNING_SESSION_PHASE_ID, spend.tokens, spend.cents);
  return null;
}
