import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
import { buildIntegrationCapabilityGroupSections } from "@/lib/capabilities";
import { CapabilitySection } from "./CapabilitySection";
import type { Dispatch, SetStateAction } from "react";
import { useMemo } from "react";
import { DEFAULT_CAPABILITY_STATE, type DisplayCapability, getCapabilityLabel } from "./lib";

export interface CapabilitiesTabProps {
  integration: OrganizationsIntegration;
  integrationDef: IntegrationsIntegrationDefinition | undefined;
  capabilityStates: Record<string, IntegrationCapabilityStateState>;
  setCapabilityStates: Dispatch<SetStateAction<Record<string, IntegrationCapabilityStateState>>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  capabilitiesMutationPending: boolean;
  onApplyCapabilityChanges: (updates: IntegrationCapabilityState[]) => void | Promise<void>;
}

export function CapabilitiesTab({
  integration,
  integrationDef,
  capabilityStates,
  setCapabilityStates,
  canUpdateIntegrations,
  permissionsLoading,
  capabilitiesMutationPending,
  onApplyCapabilityChanges,
}: CapabilitiesTabProps) {
  const capabilities = useMemo(() => {
    const byName = new Map<string, DisplayCapability>();

    (integrationDef?.capabilities || []).forEach((definition) => {
      if (!definition.name) return;
      byName.set(definition.name, {
        name: definition.name,
        definition,
        state: DEFAULT_CAPABILITY_STATE,
      });
    });

    (integration?.status?.capabilities || []).forEach((capability) => {
      if (!capability.name) return;
      const existing = byName.get(capability.name);
      byName.set(capability.name, {
        name: capability.name,
        definition: existing?.definition,
        state: capability.state || DEFAULT_CAPABILITY_STATE,
      });
    });

    return Array.from(byName.values()).sort((left, right) =>
      getCapabilityLabel(left).localeCompare(getCapabilityLabel(right)),
    );
  }, [integration?.status?.capabilities, integrationDef?.capabilities]);

  const definitionCapabilitiesSortedForGroups = useMemo(() => {
    return [...(integrationDef?.capabilities || [])]
      .filter((definition) => Boolean(definition.name))
      .sort((left, right) => left.label?.localeCompare(right.label ?? "") ?? 0);
  }, [integrationDef?.capabilities]);

  const capabilityGroupSections = useMemo(() => {
    const sections = buildIntegrationCapabilityGroupSections(integrationDef, definitionCapabilitiesSortedForGroups);
    const totalNames = sections.reduce((count, section) => count + section.names.length, 0);
    if (totalNames > 0 || capabilities.length === 0) {
      return sections;
    }
    return [
      {
        key: "all-status-only",
        label: "",
        names: [...capabilities]
          .sort((left, right) => getCapabilityLabel(left).localeCompare(getCapabilityLabel(right)))
          .map((capability) => capability.name),
      },
    ];
  }, [integrationDef, definitionCapabilitiesSortedForGroups, capabilities]);

  const capabilityByName = useMemo(() => {
    const map = new Map<string, DisplayCapability>();
    for (const capability of capabilities) {
      map.set(capability.name, capability);
    }
    return map;
  }, [capabilities]);

  const stagedCapabilityUpdates = useMemo(() => {
    return capabilities.reduce<IntegrationCapabilityState[]>((updates, capability) => {
      const serverState = capability.state || DEFAULT_CAPABILITY_STATE;
      const effectiveState = capabilityStates[capability.name] ?? serverState;
      if (effectiveState === serverState) return updates;
      if (
        effectiveState !== "STATE_ENABLED" &&
        effectiveState !== "STATE_DISABLED" &&
        effectiveState !== "STATE_REQUESTED"
      ) {
        return updates;
      }
      updates.push({ name: capability.name, state: effectiveState });
      return updates;
    }, []);
  }, [capabilities, capabilityStates]);

  const queueCapabilityStateChange = (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => {
    if (!canUpdateIntegrations || !capability.name || capabilitiesMutationPending) return;
    setCapabilityStates((previous) => ({
      ...previous,
      [capability.name!]: nextState,
    }));
  };

  if (capabilityByName.size === 0) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No capabilities available.</p>;
  }

  return (
    <>
      <div className="space-y-4">
        {capabilityGroupSections.map((section) => (
          <CapabilitySection
            key={section.key}
            section={section}
            capabilityByName={capabilityByName}
            capabilityStates={capabilityStates}
            canUpdateIntegrations={canUpdateIntegrations}
            permissionsLoading={permissionsLoading}
            capabilitiesMutationPending={capabilitiesMutationPending}
            onQueueCapabilityStateChange={queueCapabilityStateChange}
          />
        ))}
      </div>
      {stagedCapabilityUpdates.length > 0 ? (
        <div className="mt-4 flex justify-end">
          <LoadingButton
            type="button"
            color="blue"
            onClick={() => void onApplyCapabilityChanges(stagedCapabilityUpdates)}
            disabled={!canUpdateIntegrations}
            loading={capabilitiesMutationPending}
            loadingText="Updating…"
          >
            Apply changes
          </LoadingButton>
        </div>
      ) : null}
    </>
  );
}
