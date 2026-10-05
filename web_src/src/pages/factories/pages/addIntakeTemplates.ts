import { intakeSurfaceEntries, type IntakeCatalogItem } from "@/lib/intakeCatalog";

export interface AddIntakeTemplate {
  id: string;
  name: string;
  description: string;
  /** Optional integration icon. Letter glyph when omitted. */
  iconSrc?: string;
  /** True when the organization cannot create this intake yet. */
  soon?: boolean;
  /** True when the intake is in Beta for the organization. */
  beta?: boolean;
}

export const ADD_INTAKE_COPY = {
  pickerTitle: "Add intake",
  pickerDescription: "Choose a source for new backlog tasks.",
  sourceTaken: "Already set up.",
  comingSoon: "Coming soon.",
  beta: "Beta",
  loading: "Loading intakes...",
  loadFailed: "Cannot load intakes. Close this window and try again.",
} as const;

/**
 * Sources in the Add intake picker for one organization. The intake catalog
 * sets the state. Coming-soon sources stay visible and disabled.
 */
export function addIntakeTemplatesFromCatalog(catalog: readonly IntakeCatalogItem[]): AddIntakeTemplate[] {
  return intakeSurfaceEntries("addIntake", catalog).map((entry) => ({
    id: entry.key,
    name: entry.name,
    description: entry.description,
    iconSrc: entry.iconSrc,
    soon: entry.state === "soon",
    beta: entry.state === "beta",
  }));
}
