// CEL expression support for dashboard table panels.
//
// Strings wrapped in `{{ ... }}` are compiled once and evaluated per row.
// Strings without braces use legacy dot-path semantics via callers.
import { type CstNode } from "chevrotain";
import { evaluate as celEvaluate, parse as celParse } from "cel-js";

export type CompiledExpr = { ok: true; raw: string; cst: CstNode } | { ok: false; raw: string; error: string };

export type MaybeExpr = { kind: "literal"; value: string } | { kind: "expr"; expr: CompiledExpr };

export type CompiledTemplate = {
  segments: TemplateSegment[];
  hasExpr: boolean;
};

type TemplateSegment = { kind: "literal"; value: string } | { kind: "expr"; expr: CompiledExpr };

const FULL_EXPR_RE = /^\s*\{\{([\s\S]*)\}\}\s*$/;
const ANY_EXPR_RE = /\{\{([\s\S]*?)\}\}/g;

function isFullExpression(raw: string): boolean {
  return FULL_EXPR_RE.test(raw);
}

export function compileExpr(raw: string): CompiledExpr {
  const result = celParse(raw);
  if (!result.isSuccess) {
    const message = result.errors.join("; ");
    return { ok: false, raw, error: message };
  }
  return { ok: true, raw, cst: result.cst };
}

export function compileMaybeExpr(raw: string): MaybeExpr {
  if (isFullExpression(raw)) {
    const m = raw.match(FULL_EXPR_RE)!;
    return { kind: "expr", expr: compileExpr(m[1]) };
  }
  return { kind: "literal", value: raw };
}

export function compileTemplate(raw: string): CompiledTemplate {
  const segments: TemplateSegment[] = [];
  let hasExpr = false;
  let lastIndex = 0;
  ANY_EXPR_RE.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = ANY_EXPR_RE.exec(raw)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ kind: "literal", value: raw.slice(lastIndex, match.index) });
    }
    segments.push({ kind: "expr", expr: compileExpr(match[1]) });
    hasExpr = true;
    lastIndex = match.index + match[0].length;
  }
  if (lastIndex < raw.length) {
    segments.push({ kind: "literal", value: raw.slice(lastIndex) });
  }
  if (segments.length === 0) {
    segments.push({ kind: "literal", value: "" });
  }
  return { segments, hasExpr };
}

export interface ExprEnv {
  globals: Record<string, unknown>;
  functions: Record<string, CallableFunction>;
}

export function buildEnv(globals?: Record<string, unknown>): ExprEnv {
  const merged: Record<string, unknown> = {
    now: Math.floor(Date.now() / 1000),
    ...(globals ?? {}),
  };
  return { globals: merged, functions: BUILTIN_FUNCTIONS };
}

const BUILTIN_FUNCTIONS: Record<string, CallableFunction> = {
  int: toInt,
  float: toFloat,
  string: toStringValue,
  contains: (s: unknown, sub: unknown) => typeof s === "string" && typeof sub === "string" && s.includes(sub),
  startsWith: (s: unknown, p: unknown) => typeof s === "string" && typeof p === "string" && s.startsWith(p),
  endsWith: (s: unknown, p: unknown) => typeof s === "string" && typeof p === "string" && s.endsWith(p),
  matches: (s: unknown, re: unknown) => {
    if (typeof s !== "string" || typeof re !== "string") return false;
    try {
      return new RegExp(re).test(s);
    } catch {
      return false;
    }
  },
  lower: (s: unknown) => (s == null ? "" : String(s).toLowerCase()),
  upper: (s: unknown) => (s == null ? "" : String(s).toUpperCase()),
  duration: (seconds: unknown) => formatDurationSeconds(Number(seconds)),
  timestamp: (seconds: unknown) => formatTimestampSeconds(Number(seconds)),
};

function toInt(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? Math.trunc(value) : 0;
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string") {
    const n = Number(value);
    return Number.isFinite(n) ? Math.trunc(n) : 0;
  }
  return 0;
}

function toFloat(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string") {
    const n = Number(value);
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
}

function toStringValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function formatDurationSeconds(value: number): string {
  if (!Number.isFinite(value)) return "";
  const total = Math.max(0, Math.trunc(value));
  if (total < 60) return `${total}s`;
  const minutes = Math.floor(total / 60);
  if (minutes < 60) {
    const remSeconds = total % 60;
    return remSeconds === 0 ? `${minutes}m` : `${minutes}m ${remSeconds}s`;
  }
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  return remMinutes === 0 ? `${hours}h` : `${hours}h ${remMinutes}m`;
}

function formatTimestampSeconds(value: number): string {
  if (!Number.isFinite(value)) return "";
  const ms = Math.trunc(value) * 1000;
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) return "";
  return date.toISOString();
}

export function evalExpr(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): unknown {
  if (!compiled.ok) return undefined;
  const vars = { ...env.globals, ...row };
  try {
    return celEvaluate(compiled.cst, vars, env.functions);
  } catch {
    return undefined;
  }
}

export function evalRowField(
  maybe: MaybeExpr,
  row: Record<string, unknown>,
  env: ExprEnv,
  resolveLiteral: (row: Record<string, unknown>, path: string) => unknown,
): unknown {
  if (maybe.kind === "literal") return resolveLiteral(row, maybe.value);
  return evalExpr(maybe.expr, row, env);
}

export function evalTemplate(
  template: CompiledTemplate,
  row: Record<string, unknown>,
  env: ExprEnv,
  stringify: (value: unknown) => string,
): string {
  let out = "";
  for (const seg of template.segments) {
    if (seg.kind === "literal") {
      out += seg.value;
      continue;
    }
    const value = evalExpr(seg.expr, row, env);
    if (value === undefined) continue;
    out += stringify(value);
  }
  return out;
}

export function evalBoolExpr(raw: string | undefined, row: Record<string, unknown>, env: ExprEnv): boolean {
  if (!raw || !raw.trim()) return true;
  const trimmed = raw.trim();
  if (isFullExpression(trimmed)) {
    const maybe = compileMaybeExpr(trimmed);
    if (maybe.kind === "expr") {
      const result = evalExpr(maybe.expr, row, env);
      return Boolean(result);
    }
  }
  if (ANY_EXPR_RE.test(trimmed)) {
    return Boolean(evalTemplate(compileTemplate(trimmed), row, env, (v) => String(v ?? "")));
  }
  return false;
}
