import { buildEnv, compileMaybeExpr, evalRowField } from "./celExpr";
import { getValueAtPath } from "./fieldPath";

/** Resolve a column field (literal path or `{{ cel }}`) for a memory/execution row. */
export function resolveCellValue(field: string, row: unknown): unknown {
  if (!field.trim()) return undefined;
  const record = row && typeof row === "object" && !Array.isArray(row) ? (row as Record<string, unknown>) : {};
  const env = buildEnv();
  const maybe = compileMaybeExpr(field);
  return evalRowField(maybe, record, env, getValueAtPath);
}
