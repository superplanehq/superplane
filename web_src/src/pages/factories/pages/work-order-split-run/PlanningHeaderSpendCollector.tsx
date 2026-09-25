import { useRef } from "react";

import type { LiveHeaderSpend } from "@/lib/overlayHeaderSpend";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import type { CreateWithAgentView } from "../createWithAgentTypes";
import { PLANNING_SESSION_PHASE_ID, planningSessionPhase } from "../planningSessionActivity";
import {
  headerSpendFromUsageSeries,
  planningHeaderSpendToReport,
  type PlanningHeaderSpendMemory,
} from "./planningHeaderSpend";
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
  saved,
}: {
  organizationId?: string;
  view: PlanningSpendView;
  saved: LiveHeaderSpend;
}) {
  const line = planningSessionPhase(view).stream[0];
  const { usageSeries } = useLiveLogStream(
    line?.executionId ?? "",
    line?.status === "running",
    terminalCommandStatus(line?.status),
    null,
    { organizationId, canvasId: view.canvasId },
  );
  const spend = useReportedPlanningSpend(view.executionId, saved, headerSpendFromUsageSeries(usageSeries));
  useReportLiveHeaderSpend(PLANNING_SESSION_PHASE_ID, spend.tokens, spend.cents);
  return null;
}

function useReportedPlanningSpend(executionId: string, saved: LiveHeaderSpend, live: LiveHeaderSpend): LiveHeaderSpend {
  const memory = useRef<PlanningHeaderSpendMemory | undefined>(undefined);
  const reported = planningHeaderSpendToReport(memory.current, executionId, saved, live);
  memory.current = reported.memory;
  return reported.spend;
}
