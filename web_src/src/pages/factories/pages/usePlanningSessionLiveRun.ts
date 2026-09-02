import { useDescribeRun } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";

import type { CreateWithAgentView } from "./createWithAgentTypes";
import { applyPlanningSessionLiveRun } from "./planningSessionView";

export function usePlanningSessionLiveRun(
  organizationId: string | undefined,
  view: CreateWithAgentView,
): CreateWithAgentView {
  const canvasId = view.canvasId;
  const runId = view.canvasRunId || null;
  const enabled = Boolean(organizationId && canvasId && runId);
  useCanvasRuntimeWebsocket(canvasId, organizationId ?? "", enabled);
  const runQuery = useDescribeRun(canvasId, runId, enabled);
  return applyPlanningSessionLiveRun(view, runQuery.data?.run);
}
