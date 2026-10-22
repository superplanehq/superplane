import type { FactoryOnboardingAgentHarness } from "@/api-client";

import {
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  type SelectableLLMModel,
} from "@/lib/selectableLLMModels";

export type AgentModelSource = "hosted" | "anthropic" | "openai" | "openrouter" | "custom";

export type AgentModelProviderConnection = {
  provider: Exclude<AgentModelSource, "hosted">;
  connected: boolean;
  integrationId?: string;
};

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
  integrationId: string | undefined,
  connections: AgentModelProviderConnection[],
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

  const provider = connectedProvider(harness, integrationId, connections);
  if (harness === "AGENT_HARNESS_CLAUDE_CODE") {
    if (provider === "anthropic" || provider === "openrouter" || provider === "custom") {
      return provider;
    }
    return null;
  }
  return provider ?? "hosted";
}

function connectedProvider(
  harness: FactoryOnboardingAgentHarness | undefined,
  integrationId: string | undefined,
  connections: AgentModelProviderConnection[],
): AgentModelProviderConnection["provider"] | null {
  const savedId = integrationId?.trim() ?? "";
  if (savedId) {
    const match = connections.find((connection) => connection.connected && connection.integrationId === savedId);
    if (match) {
      return match.provider;
    }
  }
  if (harness === "AGENT_HARNESS_CLAUDE_CODE") {
    return firstConnected(connections, ["anthropic", "openrouter", "custom"]);
  }
  return firstConnected(connections, ["anthropic", "openai", "openrouter", "custom"]);
}

function firstConnected(
  connections: AgentModelProviderConnection[],
  order: AgentModelProviderConnection["provider"][],
): AgentModelProviderConnection["provider"] | null {
  for (const provider of order) {
    if (connections.some((connection) => connection.connected && connection.provider === provider)) {
      return provider;
    }
  }
  return null;
}
