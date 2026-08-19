import type { OrganizationsIntegration } from "@/api-client";

export type IntegrationSelection = {
  id: string;
  name: string;
  /** True only when the selected instance is ready for install/wiring. */
  ready: boolean;
};

export type IntegrationSelections = Record<string, IntegrationSelection>;

export type IntegrationInstanceSummary = {
  name: string;
  allInstances: OrganizationsIntegration[];
  readyInstances: OrganizationsIntegration[];
};

export type HomeIntegrationStatusKind = "ready" | "pending" | "error" | "none";

export type HomeIntegrationStatus = {
  kind: HomeIntegrationStatusKind;
  label: string;
  /** Instance to open in Configure when kind is pending or error. */
  configureId?: string;
};

export function selectionFromInstance(instance: OrganizationsIntegration): IntegrationSelection | null {
  const id = instance.metadata?.id;
  const name = instance.metadata?.name;
  if (!id || !name) return null;
  return { id, name, ready: instance.status?.state === "ready" };
}

/** True when every required integration has a ready selection. */
export function requiredIntegrationsReady(integrations: string[], selections: IntegrationSelections): boolean {
  return integrations.length === 0 || integrations.every((name) => selections[name]?.ready);
}

function statusForInstance(instance: OrganizationsIntegration): HomeIntegrationStatus | null {
  const id = instance.metadata?.id;
  if (!id) return null;
  if (instance.status?.state === "ready") return { kind: "ready", label: "Connected" };
  if (instance.status?.state === "error") return { kind: "error", label: "Error", configureId: id };
  return { kind: "pending", label: "Pending", configureId: id };
}

/**
 * Derives Connected / Pending / Error / Not connected.
 * When a selection is present, status reflects that instance so a preferred
 * pending connect is not masked by another ready instance.
 */
export function resolveHomeIntegrationStatus(
  data: IntegrationInstanceSummary,
  selectedId?: string,
): HomeIntegrationStatus {
  if (selectedId) {
    const selected = data.allInstances.find((instance) => instance.metadata?.id === selectedId);
    const selectedStatus = selected ? statusForInstance(selected) : null;
    if (selectedStatus) return selectedStatus;
  }

  if (data.readyInstances.length > 0) {
    return { kind: "ready", label: "Connected" };
  }

  for (const state of ["pending", "error"] as const) {
    const match = data.allInstances.find((instance) => instance.status?.state === state);
    const status = match ? statusForInstance(match) : null;
    if (status) return status;
  }

  const incomplete = data.allInstances.find((instance) => instance.metadata?.id);
  return incomplete
    ? (statusForInstance(incomplete) ?? { kind: "none", label: "Not connected" })
    : { kind: "none", label: "Not connected" };
}

function applyPreferredInstance(
  data: IntegrationInstanceSummary,
  next: IntegrationSelections,
  preferredId: string,
): boolean {
  const preferred = data.allInstances.find((instance) => instance.metadata?.id === preferredId);
  if (!preferred) return false;
  const selection = selectionFromInstance(preferred);
  if (!selection) return false;
  const current = next[data.name];
  if (current?.id === selection.id && current.ready === selection.ready) return false;
  next[data.name] = selection;
  return true;
}

function selectionNeedsUpdate(current: IntegrationSelection | undefined, nextSelection: IntegrationSelection): boolean {
  return !current || current.id !== nextSelection.id || current.ready !== nextSelection.ready;
}

/**
 * Clears selections pointing to non-ready instances and auto-selects
 * the first ready instance for unselected types. When a preferred instance
 * id is set (e.g. after Connect new), that instance is kept even while
 * pending and wins once ready. Returns updated selections if anything
 * changed, or null if no changes needed.
 */
export function syncSelectionsWithInstances(
  integrationData: IntegrationInstanceSummary[],
  selections: IntegrationSelections,
  preferredInstanceIds: Record<string, string> = {},
): IntegrationSelections | null {
  let changed = false;
  const next = { ...selections };

  for (const data of integrationData) {
    const preferredId = preferredInstanceIds[data.name];
    if (preferredId) {
      if (applyPreferredInstance(data, next, preferredId)) changed = true;
      continue;
    }

    if (next[data.name]) {
      const selected = data.allInstances.find((i) => i.metadata?.id === next[data.name].id);
      if (!selected || selected.status?.state !== "ready") {
        delete next[data.name];
        changed = true;
      } else {
        const selection = selectionFromInstance(selected);
        if (selection && selectionNeedsUpdate(next[data.name], selection)) {
          next[data.name] = selection;
          changed = true;
        }
      }
    }

    if (!next[data.name] && data.readyInstances.length > 0) {
      const first = data.readyInstances[0];
      const selection = selectionFromInstance(first);
      if (selection) {
        next[data.name] = selection;
        changed = true;
      }
    }
  }

  return changed ? next : null;
}
