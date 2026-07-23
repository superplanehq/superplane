import type { OrganizationsIntegration } from "@/api-client";

export type IntegrationSelections = Record<string, { id: string; name: string }>;

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

/** Derives Connected / Pending / Error / Not connected from instances, not only ready selections. */
export function resolveHomeIntegrationStatus(data: IntegrationInstanceSummary): HomeIntegrationStatus {
  if (data.readyInstances.length > 0) {
    return { kind: "ready", label: "Connected" };
  }

  const pending = data.allInstances.find((instance) => instance.status?.state === "pending");
  if (pending?.metadata?.id) {
    return { kind: "pending", label: "Pending", configureId: pending.metadata.id };
  }

  const errored = data.allInstances.find((instance) => instance.status?.state === "error");
  if (errored?.metadata?.id) {
    return { kind: "error", label: "Error", configureId: errored.metadata.id };
  }

  const incomplete = data.allInstances.find((instance) => instance.metadata?.id);
  if (incomplete?.metadata?.id) {
    return { kind: "pending", label: "Pending", configureId: incomplete.metadata.id };
  }

  return { kind: "none", label: "Not connected" };
}

/**
 * Clears selections pointing to non-ready instances and auto-selects
 * the first ready instance for unselected types. Returns updated
 * selections if anything changed, or null if no changes needed.
 */
export function syncSelectionsWithInstances(
  integrationData: IntegrationInstanceSummary[],
  selections: IntegrationSelections,
): IntegrationSelections | null {
  let changed = false;
  const next = { ...selections };

  for (const data of integrationData) {
    if (next[data.name]) {
      const selected = data.allInstances.find((i) => i.metadata?.id === next[data.name].id);
      if (selected && selected.status?.state !== "ready") {
        delete next[data.name];
        changed = true;
      }
    }

    if (!next[data.name] && data.readyInstances.length > 0) {
      const first = data.readyInstances[0];
      if (first.metadata?.id && first.metadata?.name) {
        next[data.name] = { id: first.metadata.id, name: first.metadata.name };
        changed = true;
      }
    }
  }

  return changed ? next : null;
}
