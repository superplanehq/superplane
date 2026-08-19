/**
 * Turns a machine-style line or phase name into readable title copy.
 * `plan-and-implement` → `Plan and Implement`.
 */
export function humanizeLineName(name: string | undefined | null): string {
  const trimmed = name?.trim();
  if (!trimmed) {
    return "Unnamed line";
  }

  const smallWords = new Set(["and", "or", "of", "the", "to", "for", "in", "on", "a", "an"]);
  const words = trimmed.split(/[-_\s]+/).filter(Boolean);
  return words
    .map((word, index) => {
      const lower = word.toLowerCase();
      if (index > 0 && smallWords.has(lower)) {
        return lower;
      }
      return lower.charAt(0).toUpperCase() + lower.slice(1);
    })
    .join(" ");
}

/**
 * Short line description when the API has no description field:
 * automation names joined as a flow, e.g. `Refund Planner → Refund Implementer`.
 */
export function formatLinePhaseDescription(
  steps: Array<{ app?: { app?: string } }> | undefined | null,
  apps: Array<{ id?: string; name?: string }> | undefined | null,
): string | undefined {
  const names = (steps ?? [])
    .map((step) => {
      const appId = step.app?.app?.trim();
      return apps?.find((app) => app.id === appId)?.name?.trim();
    })
    .filter((name): name is string => Boolean(name));

  if (names.length === 0) {
    return undefined;
  }
  return names.join(" → ");
}
