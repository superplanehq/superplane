import { agentRunNewTokenCount } from "@/lib/agentRunTelemetryChart";
import type { AgentPromptUsageSeries } from "@/lib/agentRunTelemetry";
import { EMPTY_LIVE_HEADER_SPEND, type LiveHeaderSpend } from "@/lib/overlayHeaderSpend";

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
  let usd = 0;
  for (const prompt of series) {
    tokens += agentRunNewTokenCount(prompt.telemetry);
    const promptUsd = prompt.telemetry.usage.total_cost_usd;
    if (promptUsd != null && Number.isFinite(promptUsd)) {
      usd += promptUsd;
    }
  }
  return { tokens, cents: Math.round(usd * 100) };
}

export type PlanningHeaderSpendMemory = {
  executionId: string;
  earlier: LiveHeaderSpend;
};

export function planningHeaderSpendToReport(
  memory: PlanningHeaderSpendMemory | undefined,
  executionId: string,
  saved: LiveHeaderSpend,
  live: LiveHeaderSpend,
): { memory: PlanningHeaderSpendMemory; spend: LiveHeaderSpend } {
  const next = rememberEarlierSpend(memory, executionId, saved, live);
  return {
    memory: next,
    spend: addSpend(next.earlier, live),
  };
}

function rememberEarlierSpend(
  memory: PlanningHeaderSpendMemory | undefined,
  executionId: string,
  saved: LiveHeaderSpend,
  live: LiveHeaderSpend,
): PlanningHeaderSpendMemory {
  if (!memory || memory.executionId !== executionId) {
    return { executionId, earlier: earlierSpend(saved, live) };
  }
  if (savedHasMoreEarlierUsage(memory.earlier, saved, live)) {
    return { executionId, earlier: earlierSpend(saved, live) };
  }
  return memory;
}

function earlierSpend(saved: LiveHeaderSpend, live: LiveHeaderSpend): LiveHeaderSpend {
  if (saved.tokens > 0 && saved.tokens === live.tokens) {
    return EMPTY_LIVE_HEADER_SPEND;
  }
  return saved;
}

function savedHasMoreEarlierUsage(earlier: LiveHeaderSpend, saved: LiveHeaderSpend, live: LiveHeaderSpend): boolean {
  return saved.tokens > earlier.tokens + live.tokens;
}

function addSpend(left: LiveHeaderSpend, right: LiveHeaderSpend): LiveHeaderSpend {
  return { tokens: left.tokens + right.tokens, cents: left.cents + right.cents };
}
