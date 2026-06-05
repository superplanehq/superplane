import { buildEnv, compileMaybeExpr, evalRowField, type ExprEnv } from "./celExpr";
import { getValueAtPath } from "./fieldPath";
import type { WidgetTableFilter } from "./types";

function stringifyCell(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function evalCondition(row: Record<string, unknown>, filter: WidgetTableFilter, env: ExprEnv): boolean {
  const fieldMaybe = compileMaybeExpr(filter.field);
  const raw = evalRowField(fieldMaybe, row, env, getValueAtPath);
  const has = raw !== undefined;
  const val = raw == null ? "" : typeof raw === "string" ? raw : stringifyCell(raw);

  if (filter.op === "exists") return val !== "";
  if (filter.op === "not_exists") return val === "";
  if (!has) return false;

  const expected = resolveExpectedValue(row, filter.value, env);
  return compareFilterValue(filter.op, val, expected);
}

function resolveExpectedValue(row: Record<string, unknown>, rawValue: string | undefined, env: ExprEnv): string {
  if (rawValue == null || rawValue === "") return "";

  const valueMaybe = compileMaybeExpr(rawValue);
  if (valueMaybe.kind === "literal") return valueMaybe.value;

  const expectedRaw = evalRowField(valueMaybe, row, env, getValueAtPath);
  return expectedRaw == null ? "" : typeof expectedRaw === "string" ? expectedRaw : stringifyCell(expectedRaw);
}

function compareFilterValue(op: WidgetTableFilter["op"], val: string, expected: string): boolean {
  switch (op) {
    case "eq":
      return val === expected;
    case "neq":
      return val !== expected;
    case "contains":
      return val.includes(expected);
    case "not_contains":
      return !val.includes(expected);
    case "gt":
    case "lt": {
      const a = parseFloat(val);
      const b = parseFloat(expected);
      if (Number.isNaN(a) || Number.isNaN(b)) return false;
      return op === "gt" ? a > b : a < b;
    }
    default:
      return true;
  }
}

export function applyTableWhere<T extends Record<string, unknown>>(
  rows: T[],
  where: WidgetTableFilter[] | undefined,
): T[] {
  if (!where || where.length === 0) return rows;
  const env = buildEnv();
  return rows.filter((row) => where.every((cond) => evalCondition(row, cond, env)));
}
