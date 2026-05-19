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

function evalCondition(row: Record<string, unknown>, filter: WidgetTableFilter, env: ExprEnv): boolean {
  const fieldMaybe = compileMaybeExpr(filter.field);
  const raw = evalRowField(fieldMaybe, row, env, getValueAtPath);
  const has = raw !== undefined;
  const val = raw == null ? "" : typeof raw === "string" ? raw : stringifyCell(raw);

  if (filter.op === "exists") return val !== "";
  if (filter.op === "not_exists") return val === "";

  if (!has) return false;

  let expected: string;
  if (filter.value == null || filter.value === "") {
    expected = "";
  } else {
    const valueMaybe = compileMaybeExpr(filter.value);
    if (valueMaybe.kind === "literal") {
      expected = valueMaybe.value;
    } else {
      const expectedRaw = evalRowField(valueMaybe, row, env, getValueAtPath);
      expected = expectedRaw == null ? "" : typeof expectedRaw === "string" ? expectedRaw : stringifyCell(expectedRaw);
    }
  }

  switch (filter.op) {
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
      return filter.op === "gt" ? a > b : a < b;
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
