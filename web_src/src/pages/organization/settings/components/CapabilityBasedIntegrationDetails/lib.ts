import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationNodeRef,
  IntegrationsCapabilityDefinition,
  IntegrationsIntegrationDefinition,
} from "@/api-client";

export const DEFAULT_CAPABILITY_STATE: IntegrationCapabilityStateState = "STATE_UNAVAILABLE";

export type DisplayCapability = {
  name: string;
  definition?: IntegrationsCapabilityDefinition;
  state: IntegrationCapabilityStateState;
};

/** Same chip styling as inline `code` in Integration setup instructions markdown. */
export const INTEGRATION_INLINE_CODE_CLASSES = "rounded bg-black/10 px-1.5 py-0.5 font-mono text-xs";

export function getCapabilityLabel(capability: DisplayCapability): string {
  return capability.definition?.label || capability.definition?.name || capability.name || "Unnamed capability";
}

export function getCapabilityDescription(capability: DisplayCapability): string | undefined {
  return capability.definition?.description;
}

export function getCapabilityStatusLabel(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "Enabled";
    case "STATE_DISABLED":
      return "Disabled";
    case "STATE_REQUESTED":
      return "Requested";
    case "STATE_AVAILABLE":
      return "Available";
    case "STATE_UNAVAILABLE":
      return "Unavailable";
  }
}

/** Outline badge coloring aligned with former status-dot semantics (enabled=green, disabled=red, …). */
export function getCapabilityStatusBadgeClassName(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-950/50 dark:text-green-300";
    case "STATE_DISABLED":
      return "border-red-200 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300";
    case "STATE_REQUESTED":
      return "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200";
    case "STATE_AVAILABLE":
      return "border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200";
    case "STATE_UNAVAILABLE":
      return "border-gray-200 bg-gray-100 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300";
  }
}

/** Solid circle inside the status badge (matches former standalone dot colors). */
export function getCapabilityStatusBadgeDotClassName(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "bg-green-500 dark:bg-green-400";
    case "STATE_DISABLED":
      return "bg-red-500 dark:bg-red-400";
    case "STATE_REQUESTED":
      return "bg-amber-500 dark:bg-amber-400";
    case "STATE_AVAILABLE":
      return "bg-sky-500 dark:bg-sky-400";
    case "STATE_UNAVAILABLE":
      return "bg-gray-400 dark:bg-gray-500";
  }
}

export const getActiveTabClass = (activeTab?: boolean) => {
  return activeTab
    ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
    : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300";
};

export type WorkflowGroup = {
  canvasId: string;
  canvasName: string;
  nodes: Array<{ nodeId: string; nodeName: string }>;
};

export const groupNodeRefsByCanvas = (nodeRefs: IntegrationNodeRef[]): WorkflowGroup[] => {
  if (!nodeRefs) return [];

  const groups = new Map<string, WorkflowGroup>();

  nodeRefs.forEach((nodeRef) => {
    const canvasId = nodeRef.canvasId || "";
    const canvasName = nodeRef.canvasName || canvasId;
    const nodeId = nodeRef.nodeId || "";
    const nodeName = nodeRef.nodeName || nodeId;

    if (!groups.has(canvasId)) {
      groups.set(canvasId, { canvasId, canvasName, nodes: [] });
    }

    groups.get(canvasId)?.nodes.push({ nodeId, nodeName });
  });

  return Array.from(groups.entries()).map(([canvasId, data]) => ({
    canvasId,
    canvasName: data.canvasName,
    nodes: data.nodes,
  }));
};

/** Merged capability definitions plus integration status for the capabilities tab. */
export function buildCapabilitiesTabDisplayList(
  integrationDef: IntegrationsIntegrationDefinition | undefined,
  statusCapabilities: IntegrationCapabilityState[] | undefined,
): DisplayCapability[] {
  const byName = new Map<string, DisplayCapability>();

  (integrationDef?.capabilities || []).forEach((definition) => {
    if (!definition.name) return;
    byName.set(definition.name, {
      name: definition.name,
      definition,
      state: DEFAULT_CAPABILITY_STATE,
    });
  });

  (statusCapabilities || []).forEach((capability) => {
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
}

export function computeStagedCapabilityUpdates(
  capabilities: DisplayCapability[],
  capabilityStates: Record<string, IntegrationCapabilityStateState>,
): IntegrationCapabilityState[] {
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
}

/** Capability names that appear as workflow nodes when disabling those capabilities (live canvas usage). */
export function findCapabilityNamesInUseWhenDisabling(
  updates: IntegrationCapabilityState[],
  usedIn: IntegrationNodeRef[] | undefined,
): string[] {
  const disablingNames = new Set<string>();
  for (const update of updates) {
    if (update.state === "STATE_DISABLED" && update.name) {
      disablingNames.add(update.name);
    }
  }
  if (disablingNames.size === 0 || !usedIn?.length) {
    return [];
  }

  const referenced = new Set<string>();
  for (const ref of usedIn) {
    const component = ref.component?.trim();
    if (component && disablingNames.has(component)) {
      referenced.add(component);
    }
  }

  return [...referenced];
}

export type CapabilityDisableCanvasSummary = {
  canvasId: string;
  canvasName: string;
};

export type CapabilityDisableCanvasRow = {
  capabilityName: string;
  canvases: CapabilityDisableCanvasSummary[];
};

function canvasesForCapabilityInUse(
  capabilityName: string,
  usedIn: IntegrationNodeRef[],
): CapabilityDisableCanvasSummary[] {
  const canvasLabelById = new Map<string, string>();
  for (const ref of usedIn) {
    const component = ref.component?.trim();
    if (component !== capabilityName) {
      continue;
    }
    const canvasId = ref.canvasId?.trim() ?? "";
    if (!canvasId) {
      continue;
    }
    const label = (ref.canvasName?.trim() || canvasId || "").trim() || "Unknown canvas";
    if (!canvasLabelById.has(canvasId)) {
      canvasLabelById.set(canvasId, label);
    }
  }
  if (canvasLabelById.size === 0) {
    return [];
  }
  return [...canvasLabelById.entries()]
    .map(([id, canvasName]) => ({ canvasId: id, canvasName }))
    .sort((left, right) => left.canvasName.localeCompare(right.canvasName));
}

/** One row per capability in `capabilityNames`, with deduped canvases where that capability appears. */
export function buildCapabilityDisableCanvasRows(
  capabilityNames: string[],
  usedIn: IntegrationNodeRef[] | undefined,
): CapabilityDisableCanvasRow[] {
  if (!usedIn?.length || capabilityNames.length === 0) {
    return [];
  }

  const sortedNames = [...capabilityNames].sort((left, right) => left.localeCompare(right));
  const rows: CapabilityDisableCanvasRow[] = [];

  for (const capabilityName of sortedNames) {
    const canvases = canvasesForCapabilityInUse(capabilityName, usedIn);
    if (canvases.length === 0) {
      continue;
    }
    rows.push({
      capabilityName,
      canvases,
    });
  }

  return rows;
}
