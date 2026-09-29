import type { FactoryOnboardingAgentHarness } from "@/api-client";

export type WorkspaceModelSource = "hosted" | "own-key";

/**
 * Workspace setup saves the model source as the agent harness. Workspaces
 * without a saved harness use a connected key when one exists.
 */
export function workspaceModelSource(
  harness: FactoryOnboardingAgentHarness | undefined,
  anyProviderConnected: boolean,
): WorkspaceModelSource {
  if (harness === "AGENT_HARNESS_SUPERPLANE") return "hosted";
  if (harness && harness !== "AGENT_HARNESS_UNSPECIFIED") return "own-key";
  return anyProviderConnected ? "own-key" : "hosted";
}
