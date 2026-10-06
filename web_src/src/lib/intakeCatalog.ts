import { findIntakePresentation, INTAKE_PRESENTATION, intakeSurfaces, type IntakeSurface } from "./intakePresentation";

export const INTAKE_STATUSES = ["planned", "alpha", "beta", "ga", "deprecated"] as const;
export type IntakeStatus = (typeof INTAKE_STATUSES)[number];

export const INTAKE_CATEGORIES = [
  "issue_tracking",
  "error_tracking",
  "incident_management",
  "security_alerts",
  "repository_provider",
] as const;
export type IntakeCategory = (typeof INTAKE_CATEGORIES)[number];

/** What one company sees for one intake on a surface. */
export type IntakeSurfaceState = "available" | "beta" | "soon" | "hidden";

/** The fields of a catalog entry that decide what a company sees. */
export interface IntakeCatalogItem {
  key: string;
  name: string;
  category: string;
  status: string;
  available: boolean;
}

const STATE_ORDER: Record<IntakeSurfaceState, number> = { available: 0, beta: 1, soon: 2, hidden: 3 };

/**
 * The state that a company sees. A company that cannot use an Internal or a
 * Deprecated intake does not see it. Other intakes it cannot use show as
 * Coming soon.
 */
export function intakeSurfaceState(
  item: Pick<IntakeCatalogItem, "status" | "available"> | undefined,
): IntakeSurfaceState {
  if (!item) {
    return "soon";
  }
  if (item.available) {
    return item.status === "beta" ? "beta" : "available";
  }
  if (item.status === "alpha" || item.status === "deprecated") {
    return "hidden";
  }
  return "soon";
}

export interface IntakeSurfaceEntry {
  key: string;
  name: string;
  description: string;
  iconSrc?: string;
  state: Exclude<IntakeSurfaceState, "hidden">;
}

/**
 * Entries that a surface lists for a company, in display order: available,
 * then Beta, then Coming soon. Hidden entries are left out.
 */
export function intakeSurfaceEntries(
  surface: IntakeSurface,
  catalog: readonly IntakeCatalogItem[],
): IntakeSurfaceEntry[] {
  const catalogByKey = new Map(catalog.map((item) => [item.key, item]));
  const candidates: Omit<IntakeSurfaceEntry, "state">[] = [];

  for (const presentation of INTAKE_PRESENTATION) {
    if (!presentation.surfaces.includes(surface)) {
      continue;
    }
    candidates.push({
      key: presentation.key,
      name: catalogByKey.get(presentation.key)?.name || presentation.name,
      description: presentation.description,
      iconSrc: presentation.iconSrc,
    });
  }
  for (const item of catalog) {
    if (findIntakePresentation(item.key) || !intakeSurfaces(item.key, item.category).includes(surface)) {
      continue;
    }
    candidates.push({ key: item.key, name: item.name, description: "" });
  }

  const entries: IntakeSurfaceEntry[] = [];
  for (const candidate of candidates) {
    const state = intakeSurfaceState(catalogByKey.get(candidate.key));
    if (state === "hidden") {
      continue;
    }
    entries.push({ ...candidate, state });
  }

  return entries
    .map((entry, index) => ({ entry, index }))
    .sort((a, b) => STATE_ORDER[a.entry.state] - STATE_ORDER[b.entry.state] || a.index - b.index)
    .map(({ entry }) => entry);
}

export function isIntakeSelectable(state: IntakeSurfaceState | undefined): boolean {
  return state === "available" || state === "beta";
}
