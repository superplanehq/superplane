import { parsePath } from "./fieldPath";

/** Set a string leaf at a dot-path (e.g. `data.issue.number`). */
export function setNestedString(root: Record<string, unknown>, path: string, value: string): void {
  const segments = parsePath(path);
  if (segments.length === 0) return;
  let cursor: Record<string, unknown> = root;
  for (let i = 0; i < segments.length - 1; i += 1) {
    const key = segments[i]!;
    const next = cursor[key];
    if (!next || typeof next !== "object" || Array.isArray(next)) {
      const created: Record<string, unknown> = {};
      cursor[key] = created;
      cursor = created;
    } else {
      cursor = next as Record<string, unknown>;
    }
  }
  cursor[segments[segments.length - 1]!] = value;
}

export function deepMergeObjects(
  base: Record<string, unknown>,
  patch: Record<string, unknown>,
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...base };
  for (const [key, value] of Object.entries(patch)) {
    const existing = out[key];
    if (
      existing &&
      typeof existing === "object" &&
      !Array.isArray(existing) &&
      value &&
      typeof value === "object" &&
      !Array.isArray(value)
    ) {
      out[key] = deepMergeObjects(existing as Record<string, unknown>, value as Record<string, unknown>);
    } else {
      out[key] = value;
    }
  }
  return out;
}
