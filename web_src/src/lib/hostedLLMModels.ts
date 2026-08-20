export function compareModelLabels(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
}

export function uniqueSortedModelIds(ids: string[]): string[] {
  const unique = Array.from(new Set(ids.map((id) => id.trim()).filter((id) => id !== "")));
  unique.sort(compareModelLabels);
  return unique;
}

export function filterModelIds(ids: string[], query: string): string[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return ids;
  }

  return ids.filter((id) => id.toLowerCase().includes(needle));
}
