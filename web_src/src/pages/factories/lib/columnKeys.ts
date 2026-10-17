export type ColumnKey = "backlog" | `phase-${number}` | "verify" | "done";

const PHASE_KEY_PATTERN = /^phase-(\d+)$/;

export function isColumnKey(value: string | null | undefined): value is ColumnKey {
  if (!value) {
    return false;
  }
  return value === "backlog" || value === "verify" || value === "done" || PHASE_KEY_PATTERN.test(value);
}

export function phaseIndexFromColumnKey(key: ColumnKey): number | undefined {
  const match = PHASE_KEY_PATTERN.exec(key);
  if (!match) {
    return undefined;
  }
  return Number.parseInt(match[1] ?? "", 10);
}
