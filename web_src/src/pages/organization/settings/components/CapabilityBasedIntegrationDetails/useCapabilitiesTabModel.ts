import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { buildIntegrationCapabilityGroupSections } from "@/lib/capabilities";
import type { Dispatch, SetStateAction } from "react";
import { useCallback, useMemo, useState } from "react";
import {
  buildCapabilitiesTabDisplayList,
  buildCapabilityDisableCanvasRows,
  computeStagedCapabilityUpdates,
  type DisplayCapability,
  findCapabilityNamesInUseWhenDisabling,
  getCapabilityLabel,
} from "./lib";

type UseCapabilitiesTabModelArgs = {
  integration: OrganizationsIntegration;
  integrationDef: IntegrationsIntegrationDefinition | undefined;
  capabilityStates: Record<string, IntegrationCapabilityStateState>;
  setCapabilityStates: Dispatch<SetStateAction<Record<string, IntegrationCapabilityStateState>>>;
  canUpdateIntegrations: boolean;
  capabilitiesMutationPending: boolean;
  onApplyCapabilityChanges: (updates: IntegrationCapabilityState[]) => void | Promise<void>;
};

export function useCapabilitiesTabModel({
  integration,
  integrationDef,
  capabilityStates,
  setCapabilityStates,
  canUpdateIntegrations,
  capabilitiesMutationPending,
  onApplyCapabilityChanges,
}: UseCapabilitiesTabModelArgs) {
  const [pendingCapabilityUpdates, setPendingCapabilityUpdates] = useState<IntegrationCapabilityState[] | null>(null);
  const usedInCanvases = integration.status?.usedIn;

  const capabilities = useMemo(
    () => buildCapabilitiesTabDisplayList(integrationDef, integration?.status?.capabilities),
    [integrationDef, integration?.status?.capabilities],
  );

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

  const stagedCapabilityUpdates = useMemo(
    () => computeStagedCapabilityUpdates(capabilities, capabilityStates),
    [capabilities, capabilityStates],
  );

  const capabilityDisableCanvasRows = useMemo(() => {
    if (!pendingCapabilityUpdates?.length) return [];
    const names = findCapabilityNamesInUseWhenDisabling(pendingCapabilityUpdates, usedInCanvases);
    return buildCapabilityDisableCanvasRows(names, usedInCanvases);
  }, [pendingCapabilityUpdates, usedInCanvases]);

  const requestCapabilityUpdates = useCallback(
    (updates: IntegrationCapabilityState[]) => {
      const inUse = findCapabilityNamesInUseWhenDisabling(updates, usedInCanvases);
      if (inUse.length === 0) {
        void onApplyCapabilityChanges(updates);
        return;
      }
      setPendingCapabilityUpdates(updates);
    },
    [onApplyCapabilityChanges, usedInCanvases],
  );

  const confirmPendingCapabilityUpdates = useCallback(() => {
    if (!pendingCapabilityUpdates?.length) return;
    void onApplyCapabilityChanges(pendingCapabilityUpdates);
    setPendingCapabilityUpdates(null);
  }, [onApplyCapabilityChanges, pendingCapabilityUpdates]);

  const clearPendingCapabilityUpdates = useCallback(() => {
    setPendingCapabilityUpdates(null);
  }, []);

  const queueCapabilityStateChange = useCallback(
    (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => {
      if (!canUpdateIntegrations || !capability.name || capabilitiesMutationPending) return;
      setCapabilityStates((previous) => ({
        ...previous,
        [capability.name!]: nextState,
      }));
    },
    [canUpdateIntegrations, capabilitiesMutationPending, setCapabilityStates],
  );

  return {
    capabilityByName,
    capabilityGroupSections,
    stagedCapabilityUpdates,
    pendingCapabilityUpdates,
    capabilityDisableCanvasRows,
    requestCapabilityUpdates,
    confirmPendingCapabilityUpdates,
    clearPendingCapabilityUpdates,
    queueCapabilityStateChange,
  };
}
