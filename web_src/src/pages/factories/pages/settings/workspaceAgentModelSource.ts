import type { FactoryOnboardingAgentHarness } from "@/api-client";

import {
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  type SelectableLLMModel,
} from "@/lib/selectableLLMModels";

export type AgentModelSource = "hosted" | "anthropic" | "openai" | "openrouter" | "custom";

export function modelMatchesAgentSource(model: SelectableLLMModel, source: AgentModelSource | undefined): boolean {
  if (!source) {
    return false;
  }
  if (source === "hosted") {
    return model.source.id === SELECTABLE_LLM_SOURCE_HOSTED;
  }
  return model.source.id === SELECTABLE_LLM_SOURCE_BYOK && model.provider.id === source;
}

export function workspaceAgentModelSource(
  harness: FactoryOnboardingAgentHarness | undefined,
  reportedSource: string | undefined,
): AgentModelSource | null {
  if (harness === "AGENT_HARNESS_SUPERPLANE") {
    return "hosted";
  }
  if (harness === "AGENT_HARNESS_CODEX") {
    return "openai";
  }
  if (harness === "AGENT_HARNESS_CURSOR") {
    return null;
  }
  return parseAgentModelSource(reportedSource);
}

function parseAgentModelSource(value: string | undefined): AgentModelSource | null {
  switch (value?.trim()) {
    case "hosted":
    case "anthropic":
    case "openai":
    case "openrouter":
    case "custom":
      return value.trim() as AgentModelSource;
    default:
      return null;
  }
}
