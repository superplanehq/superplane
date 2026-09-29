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
  classifiedSavedTokens: number;
  lastLiveTokens: number;
  sawLiveGrow: boolean;
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
    return initialMemory(executionId, saved, live);
  }
  return absorbSaved(noteLiveGrowth(memory, live), saved);
}

function initialMemory(executionId: string, saved: LiveHeaderSpend, live: LiveHeaderSpend): PlanningHeaderSpendMemory {
  return {
    executionId,
    earlier: sameRunAlreadySaved(saved, live) ? EMPTY_LIVE_HEADER_SPEND : saved,
    classifiedSavedTokens: saved.tokens,
    lastLiveTokens: live.tokens,
    sawLiveGrow: false,
  };
}

function noteLiveGrowth(memory: PlanningHeaderSpendMemory, live: LiveHeaderSpend): PlanningHeaderSpendMemory {
  if (live.tokens <= memory.lastLiveTokens) {
    return memory;
  }
  return { ...memory, lastLiveTokens: live.tokens, sawLiveGrow: true };
}

function absorbSaved(memory: PlanningHeaderSpendMemory, saved: LiveHeaderSpend): PlanningHeaderSpendMemory {
  if (saved.tokens === memory.classifiedSavedTokens || currentRunWasSaved(memory, saved)) {
    return { ...memory, classifiedSavedTokens: saved.tokens };
  }
  if (saved.tokens <= memory.earlier.tokens) {
    return { ...memory, classifiedSavedTokens: saved.tokens };
  }
  return { ...memory, earlier: saved, classifiedSavedTokens: saved.tokens };
}

function sameRunAlreadySaved(saved: LiveHeaderSpend, live: LiveHeaderSpend): boolean {
  return saved.tokens > 0 && saved.tokens === live.tokens;
}

function currentRunWasSaved(memory: PlanningHeaderSpendMemory, saved: LiveHeaderSpend): boolean {
  if (memory.lastLiveTokens <= 0 || saved.tokens !== memory.earlier.tokens + memory.lastLiveTokens) {
    return false;
  }
  if (memory.classifiedSavedTokens !== memory.earlier.tokens) {
    return false;
  }
  if (memory.earlier.tokens === 0) {
    return memory.sawLiveGrow;
  }
  return true;
}

function addSpend(left: LiveHeaderSpend, right: LiveHeaderSpend): LiveHeaderSpend {
  return { tokens: left.tokens + right.tokens, cents: left.cents + right.cents };
}
