export type UnknownRecord = Record<string, unknown>;

export function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null;
}

export function asString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}
