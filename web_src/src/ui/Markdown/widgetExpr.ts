// CEL expression support for widget blocks.
//
// Strings inside widget YAML wrapped in `{{ ... }}` are compiled to a CEL CST
// once per parse cycle and evaluated against each row at render time. Strings
// without `{{ ... }}` keep their existing dot-path semantics, so all widget
// blocks authored before this lived continue to work unchanged.
//
// `cel-js` parses `expression -> CST` via `parse()` and runs `(CST, vars,
// functions) -> unknown` via `evaluate()`. Custom functions are passed via
// the third argument; `cel-js` does not support method-style calls on
// strings, so functions like `contains()` are exposed as free functions
// (`contains(s, sub)`) rather than `s.contains(sub)`.
import { type CstNode } from "chevrotain";
import { evaluate as celEvaluate, parse as celParse } from "cel-js";

//
// Compiled-expression types
//

/** A single compiled CEL expression, or a parse failure. Both shapes carry the original raw source for diagnostics. */
export type CompiledExpr = { ok: true; raw: string; cst: CstNode } | { ok: false; raw: string; error: string };

/**
 * A field/value string that is either a literal dot-path or a single CEL
 * expression. Used in places where the entire string is one or the other:
 * `column.field`, `chart.x/y/group`, `number.field`, `where[i].field`,
 * `where[i].value`, `action.show`.
 */
export type MaybeExpr = { kind: "literal"; value: string } | { kind: "expr"; expr: CompiledExpr };

/**
 * A template string that may interleave literal text with `{{ ... }}` CEL
 * segments. Used for `action.confirm` and `action.fill[*]`, where the
 * authored value is meant to be a string with embedded interpolations.
 */
export type CompiledTemplate = {
  segments: TemplateSegment[];
  /** True if at least one segment is an expression — useful for `useMemo` keys / debugging. */
  hasExpr: boolean;
};

type TemplateSegment = { kind: "literal"; value: string } | { kind: "expr"; expr: CompiledExpr };

//
// Pattern detection
//

const FULL_EXPR_RE = /^\s*\{\{([\s\S]*)\}\}\s*$/;
const ANY_EXPR_RE = /\{\{([\s\S]*?)\}\}/g;

/** True when the trimmed string is exactly one `{{ ... }}` block (no surrounding text). */
function isFullExpression(raw: string): boolean {
  return FULL_EXPR_RE.test(raw);
}

//
// Compile helpers
//

export function compileExpr(raw: string): CompiledExpr {
  const result = celParse(raw);
  if (!result.isSuccess) {
    const message = result.errors.join("; ");
    // eslint-disable-next-line no-console
    console.warn("[widget] CEL parse error:", raw, message);
    return { ok: false, raw, error: message };
  }
  return { ok: true, raw, cst: result.cst };
}

/**
 * Compile a string that's either a single `{{ expr }}` or a literal. The
 * inner expression text is whatever's between the outermost `{{` and `}}`
 * after trimming.
 */
export function compileMaybeExpr(raw: string): MaybeExpr {
  if (isFullExpression(raw)) {
    const m = raw.match(FULL_EXPR_RE)!;
    return { kind: "expr", expr: compileExpr(m[1]) };
  }
  return { kind: "literal", value: raw };
}

/**
 * Compile a template string with zero or more `{{ ... }}` CEL segments.
 * Literal text outside the braces is preserved verbatim. A string with no
 * braces returns a single literal segment.
 */
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

//
// Evaluation context
//

/** Variables and functions available to every CEL expression in the widget. */
export interface ExprEnv {
  /** Per-render globals merged into every row's variable bag. */
  globals: Record<string, unknown>;
  /** Custom functions registered with cel-js. */
  functions: Record<string, CallableFunction>;
}

/** Build a fresh evaluation env with `now` (Unix seconds) and the standard widget function set. */
export function buildEnv(globals?: Record<string, unknown>): ExprEnv {
  const merged: Record<string, unknown> = {
    now: Math.floor(Date.now() / 1000),
    ...(globals ?? {}),
  };
  return { globals: merged, functions: BUILTIN_FUNCTIONS };
}

//
// Custom function library.
//
// cel-js comes with `size()` and `has()` macros plus standard arithmetic /
// boolean / ternary / index access. Anything else commonly useful in
// widgets (numeric coercion, string casing, date formatting) is registered
// here. All functions are fail-soft: bad input returns a safe default
// instead of throwing, so a single bad row doesn't break the widget.
//

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

//
// Evaluation
//

/**
 * Run a compiled expression against a row + env. Returns `undefined` when
 * the expression failed to parse or threw at runtime. cel-js throws on
 * missing identifiers, type errors, etc.; callers decide how to react
 * (drop the row, hide the button, render empty cell, ...).
 */
export function evalExpr(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): unknown {
  if (!compiled.ok) return undefined;
  const vars = { ...env.globals, ...row };
  try {
    return celEvaluate(compiled.cst, vars, env.functions);
  } catch (err) {
    // eslint-disable-next-line no-console
    console.warn("[widget] CEL eval error:", compiled.raw, err instanceof Error ? err.message : err);
    return undefined;
  }
}

/**
 * Resolve a `MaybeExpr` against a row. Literals fall back to the legacy
 * dot-path resolver so existing widgets keep their semantics.
 */
export function evalRowField(
  maybe: MaybeExpr,
  row: Record<string, unknown>,
  env: ExprEnv,
  resolveLiteral: (row: Record<string, unknown>, path: string) => unknown,
): unknown {
  if (maybe.kind === "literal") return resolveLiteral(row, maybe.value);
  return evalExpr(maybe.expr, row, env);
}

/**
 * Render a compiled template by stringifying each expression result and
 * joining with the literal segments. Per-segment runtime errors render as
 * empty so a single bad expression doesn't blank out the surrounding text.
 */
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
