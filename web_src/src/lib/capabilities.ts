import type {
  IntegrationsCapabilityDefinition,
  IntegrationsIntegrationDefinition,
  SuperplaneActionsAction,
  TriggersTrigger,
} from "@/api-client";

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

export type CapabilityGroupSection = {
  key: string;
  label: string;
  names: string[];
};

/**
 * Groups capability names using {@link IntegrationsIntegrationDefinition.capabilityGroups} when present.
 * Otherwise returns a single section with label "" and every defined capability name.
 */
export function buildIntegrationCapabilityGroupSections(
  definition: IntegrationsIntegrationDefinition | undefined,
  defsSorted: IntegrationsCapabilityDefinition[],
): CapabilityGroupSection[] {
  const defsWithName = defsSorted.filter((def) => Boolean(def.name));
  const byName = new Map(defsWithName.map((def) => [def.name!, def]));
  const namesSortedAlphabetically = defsWithName.map((def) => def.name!);

  function sortNamesWithin(names: string[]): string[] {
    return [...names].sort((leftName, rightName) => {
      const left = byName.get(leftName);
      const right = byName.get(rightName);
      if (!left || !right) {
        return 0;
      }
      return left.label?.localeCompare(right.label ?? "") ?? 0;
    });
  }

  const groups = definition?.capabilityGroups;
  if (!groups?.length) {
    return [{ key: "all", label: "", names: sortNamesWithin(namesSortedAlphabetically) }];
  }

  const allowed = new Set(namesSortedAlphabetically);
  const seen = new Set<string>();
  const sections: CapabilityGroupSection[] = [];

  groups.forEach((group, index) => {
    const ordered = group.capabilities ?? [];
    const names = ordered.filter((name): name is string => Boolean(name) && allowed.has(name));
    names.forEach((name) => seen.add(name));
    if (names.length === 0) {
      return;
    }
    sections.push({
      key: `group-${index}`,
      label: group.label?.trim() || `Group ${index + 1}`,
      names: sortNamesWithin(names),
    });
  });

  const orphans = namesSortedAlphabetically.filter((name) => !seen.has(name));
  if (orphans.length > 0) {
    sections.push({
      key: "other",
      label: "Other",
      names: sortNamesWithin(orphans),
    });
  }

  return sections;
}
