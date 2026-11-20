export type ColumnKey = "backlog" | `phase-${number}` | `custom-${string}` | "verify" | "done";

const PHASE_KEY_PATTERN = /^phase-(\d+)$/;

export function isColumnKey(value: string | null | undefined): value is ColumnKey {
  return Boolean(
    value &&
      (value === "backlog" ||
        value === "verify" ||
        value === "done" ||
        PHASE_KEY_PATTERN.test(value) ||
        value.startsWith("custom-")),
  );
}

export function phaseIndexFromColumnKey(key: ColumnKey): number | undefined {
  const match = PHASE_KEY_PATTERN.exec(key);
  return match ? Number.parseInt(match[1] ?? "", 10) : undefined;
}
