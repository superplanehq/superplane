import type { IntegrationCapabilityStateState, IntegrationsCapabilityDefinition } from "@/api-client";

export const DEFAULT_CAPABILITY_STATE: IntegrationCapabilityStateState = "STATE_UNAVAILABLE";

export type DisplayCapability = {
  name: string;
  definition?: IntegrationsCapabilityDefinition;
  state: IntegrationCapabilityStateState;
};

export function getCapabilityLabel(capability: DisplayCapability): string {
  return capability.definition?.label || capability.definition?.name || capability.name || "Unnamed capability";
}

export function getCapabilityDescription(capability: DisplayCapability): string | undefined {
  return capability.definition?.description;
}

export function getCapabilityStatusDotClass(state: IntegrationCapabilityStateState): string {
  if (state === "STATE_ENABLED") return "bg-green-500";
  if (state === "STATE_DISABLED") return "bg-red-500";
  if (state === "STATE_REQUESTED") return "bg-amber-500";
  return "bg-gray-400 dark:bg-gray-500";
}
