import type { IntegrationsCapabilityDefinition, SuperplaneActionsAction, TriggersTrigger } from "@/api-client";

export function actionsFromCapabilities(capabilities: IntegrationsCapabilityDefinition[]): SuperplaneActionsAction[] {
  return capabilities
    .filter((capability) => capability.type === "TYPE_ACTION")
    .map((capability) => ({
      name: capability.name,
      label: capability.label,
      description: capability.description,
      configuration: capability.configuration,
      outputChannels: capability.outputChannels,
    }));
}

export function triggersFromCapabilities(capabilities: IntegrationsCapabilityDefinition[]): TriggersTrigger[] {
  return capabilities
    .filter((capability) => capability.type === "TYPE_TRIGGER")
    .map((capability) => ({
      name: capability.name,
      label: capability.label,
      description: capability.description,
      configuration: capability.configuration,
    }));
}
