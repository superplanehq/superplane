import type { FactoryOnboardingAgentHarness } from "@/api-client";

export type WorkspaceModelSource = "hosted" | "own-key" | "subscription";

/**
 * Workspace setup saves the model source as the agent harness. Workspaces
 * without a saved harness use a connected key when one exists.
 */
export function workspaceModelSource(
  harness: FactoryOnboardingAgentHarness | undefined,
  anyProviderConnected: boolean,
  codexSubscription = false,
): WorkspaceModelSource {
  if (codexSubscription) return "subscription";
  if (harness === "AGENT_HARNESS_SUPERPLANE") return "hosted";
  if (harness && harness !== "AGENT_HARNESS_UNSPECIFIED") return "own-key";
  return anyProviderConnected ? "own-key" : "hosted";
}

export function workspaceModelSourceLabel(source: WorkspaceModelSource) {
  if (source === "subscription") return "ChatGPT subscriptions";
  return source === "hosted" ? "SuperPlane" : "Your API key";
}
