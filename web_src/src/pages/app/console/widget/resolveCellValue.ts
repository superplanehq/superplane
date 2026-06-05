import { buildEnv, compileMaybeExpr, evalRowField, type ExprEnv, type MaybeExpr } from "./celExpr";
import { getValueAtPath } from "./fieldPath";

/**
 * A field reference compiled once and bound to a shared expression env so it
 * can be re-evaluated against many rows without paying CEL parse cost per
 * cell. Used by row-bound renderers (charts in particular) that walk a
 * dataset of N rows and would otherwise recompile each field N times.
 */
export interface CompiledFieldResolver {
  resolve: (row: unknown) => unknown;
}

const EMPTY_RESOLVER: CompiledFieldResolver = { resolve: () => undefined };

/**
 * Pre-compile a literal dot path or full `{{ cel }}` expression for repeated
 * evaluation. Pass a shared `env` when resolving multiple fields together so
 * they observe the same `now` and builtin context.
 */
export function compileFieldResolver(field: string, env: ExprEnv = buildEnv()): CompiledFieldResolver {
  if (!field.trim()) return EMPTY_RESOLVER;
  const maybe: MaybeExpr = compileMaybeExpr(field);
  return {
    resolve: (row: unknown) => {
      const record = row && typeof row === "object" && !Array.isArray(row) ? (row as Record<string, unknown>) : {};
      return evalRowField(maybe, record, env, getValueAtPath);
    },
  };
}

/** Resolve a column field (literal path or `{{ cel }}`) for a memory/execution row. */
export function resolveCellValue(field: string, row: unknown): unknown {
  return compileFieldResolver(field).resolve(row);
}
