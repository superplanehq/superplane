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
