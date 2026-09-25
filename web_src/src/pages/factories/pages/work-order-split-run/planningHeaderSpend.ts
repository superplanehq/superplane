import { agentRunNewTokenCount } from "@/lib/agentRunTelemetryChart";
import type { AgentPromptUsageSeries } from "@/lib/agentRunTelemetry";
import type { LiveHeaderSpend } from "@/lib/overlayHeaderSpend";

import type { CreateWithAgentView } from "../createWithAgentTypes";

export function planningHeaderSpendActive(
  footerKind: string,
  view: Pick<CreateWithAgentView, "machineStatus" | "canvasId" | "executionId">,
): boolean {
  if (footerKind !== "draft") {
    return false;
  }
  if (view.machineStatus !== "running" && view.machineStatus !== "waiting") {
    return false;
  }
  return Boolean(view.canvasId && view.executionId);
}

export function headerSpendFromUsageSeries(series: AgentPromptUsageSeries[]): LiveHeaderSpend {
  let tokens = 0;
  let cents = 0;
  for (const prompt of series) {
    tokens += agentRunNewTokenCount(prompt.telemetry);
    const usd = prompt.telemetry.usage.total_cost_usd;
    if (usd != null && Number.isFinite(usd)) {
      cents += Math.round(usd * 100);
    }
  }
  return { tokens, cents };
}
