// CEL expression support for dashboard table panels.
//
// Strings wrapped in `{{ ... }}` are compiled once and evaluated per row.
// Strings without braces use legacy dot-path semantics via callers.
import { Environment, EvaluationError, TypeError as CelTypeError } from "@marcbachmann/cel-js";

import { registerCustomFunctions } from "./celBuiltins";

type CelRun = (context: Record<string, unknown>) => unknown;

export type CompiledExpr = { ok: true; raw: string; run: CelRun } | { ok: false; raw: string; error: string };

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

/**
 * Identifier substituted for bare `$` so CEL (whose grammar only accepts
 * `[a-zA-Z_][a-zA-Z0-9_]*` identifiers) can parse canvas-style node refs
 * like `$["deploy-prod"].outputs.url`. Rows that need to resolve those
 * references must carry the same map under this key.
 */
export const DOLLAR_REWRITE_IDENTIFIER = "__runNodes__";

/**
 * Rewrite bare `$` tokens to `DOLLAR_REWRITE_IDENTIFIER` outside of string
 * literals so CEL can parse expressions like `$["node name"].outputs.x`.
 * Single- and double-quoted strings are copied verbatim (with backslash
 * escapes preserved) so a `$` inside a string literal is left alone.
 *
 * The rewrite is purely additive: CEL's lexer rejects bare `$` today, so
 * no existing expression depended on the previous behavior.
 */
export function rewriteDollarRefs(raw: string): string {
  let out = "";
  let i = 0;
  while (i < raw.length) {
    const ch = raw[i];
    if (ch === '"' || ch === "'") {
      const literal = copyStringLiteral(raw, i, ch);
      out += literal.value;
      i = literal.next;
      continue;
    }
    if (ch === "$") {
      out += DOLLAR_REWRITE_IDENTIFIER;
      i++;
      continue;
    }
    out += ch;
    i++;
  }
  return out;
}

/**
 * Copy a quoted string literal verbatim (preserving backslash escapes),
 * starting at the opening quote at `start`. Returns the copied text and the
 * index immediately after the closing quote (or end of input).
 */
function copyStringLiteral(raw: string, start: number, quote: string): { value: string; next: number } {
  let value = quote;
  let i = start + 1;
  while (i < raw.length && raw[i] !== quote) {
    if (raw[i] === "\\" && i + 1 < raw.length) {
      value += raw[i] + raw[i + 1];
      i += 2;
      continue;
    }
    value += raw[i];
    i++;
  }
  if (i < raw.length) {
    value += raw[i];
    i++;
  }
  return { value, next: i };
}

export function compileExpr(raw: string): CompiledExpr {
  const rewritten = rewriteDollarRefs(raw);
  try {
    const compiled = ENV.parse(rewritten) as unknown as CelRun;
    return { ok: true, raw, run: compiled };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { ok: false, raw, error: message };
  }
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
  /**
   * Preserved for backward compatibility with legacy callers. Custom
   * functions are now registered on the shared module-level `Environment`
   * and this field is unused at eval time.
   */
  functions: Record<string, CallableFunction>;
}

export function buildEnv(globals?: Record<string, unknown>): ExprEnv {
  const merged: Record<string, unknown> = {
    now: Math.floor(Date.now() / 1000),
    ...(globals ?? {}),
  };
  return { globals: merged, functions: {} };
}

export type EvalResult = { ok: true; value: unknown } | { ok: false; error: string };

/**
 * Evaluate a compiled CEL expression. Returns `undefined` on any error so
 * existing callers (column rendering, payload merging, etc.) keep failing
 * gracefully.
 */
export function evalExpr(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): unknown {
  const detailed = evalExprDetailed(compiled, row, env);
  return detailed.ok ? detailed.value : undefined;
}

/**
 * Same as `evalExpr` but surfaces compile/eval errors so the editor can
 * display them in inline previews.
 *
 * `@marcbachmann/cel-js` requires integer arithmetic to run on `BigInt`
 * values (`10n / 2` works, `10 / 2` throws `dyn<double> / int`), so we
 * upfront-coerce safe-integer JS numbers to BigInt before evaluation and
 * fall back to a broader retry that also converts numeric-looking strings
 * — memory rows commonly stringify scalars, and authors expect
 * `{{ value / 2 }}` to "just work" on those rows. Non-numeric strings stay
 * as strings so equality checks like `name == "42"` keep their semantics.
 * Results are normalized so safe-integer BigInts flow back out as plain
 * JS numbers for downstream formatters, aggregators, and `JSON.stringify`.
 */
export function evalExprDetailed(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): EvalResult {
  if (!compiled.ok) return { ok: false, error: compiled.error };
  const merged = { ...env.globals, ...row };
  const upfront = coerceNumbers(merged) as Record<string, unknown>;
  try {
    return { ok: true, value: normalizeCelValue(compiled.run(upfront)) };
  } catch (initial) {
    if (!isEvalRetryable(initial)) {
      return { ok: false, error: errorMessage(initial) };
    }
    const coerced = coerceNumericStrings(upfront);
    if (coerced === upfront) {
      return { ok: false, error: errorMessage(initial) };
    }
    try {
      return { ok: true, value: normalizeCelValue(compiled.run(coerced as Record<string, unknown>)) };
    } catch (retry) {
      return { ok: false, error: errorMessage(retry) };
    }
  }
}

function isEvalRetryable(err: unknown): boolean {
  return err instanceof EvaluationError || err instanceof CelTypeError;
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
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

/**
 * Render a template like `evalTemplate` but surface the first compile/eval
 * error encountered. Useful for the payload editor preview where authors
 * benefit from a concrete error string instead of silent emptiness.
 */
export function evalTemplateDetailed(
  template: CompiledTemplate,
  row: Record<string, unknown>,
  env: ExprEnv,
  stringify: (value: unknown) => string,
): { ok: true; value: string } | { ok: false; error: string } {
  let out = "";
  for (const seg of template.segments) {
    if (seg.kind === "literal") {
      out += seg.value;
      continue;
    }
    const detailed = evalExprDetailed(seg.expr, row, env);
    if (!detailed.ok) return { ok: false, error: detailed.error };
    if (detailed.value === undefined) continue;
    out += stringify(detailed.value);
  }
  return { ok: true, value: out };
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

/**
 * Recursively upgrade safe-integer JS `number` values in the context to
 * `BigInt` so CEL's int-typed operators (division, modulo, comparison with
 * int literals) can dispatch. Fractional or non-safe-integer numbers are
 * left alone so double arithmetic still works. Strings, arrays, and plain
 * objects are traversed; `Date` instances and primitives pass through
 * unchanged.
 */
function coerceNumbers(value: unknown): unknown {
  if (typeof value === "number") {
    return Number.isFinite(value) && Number.isSafeInteger(value) ? BigInt(value) : value;
  }
  if (Array.isArray(value)) {
    return value.map(coerceNumbers);
  }
  if (isPlainObject(value)) {
    const out: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(value)) {
      out[key] = coerceNumbers(val);
    }
    return out;
  }
  return value;
}

/**
 * Retry-time coercion: convert numeric-looking strings into `BigInt` (for
 * integer strings that fit safely) or `number` (for decimal strings) so
 * `{{ value / 2 }}` works on stringified memory rows. Non-numeric strings
 * are preserved verbatim to keep string equality checks like `name == "42"`
 * intact. Returns the original reference when nothing changed so callers
 * can short-circuit.
 */
function coerceNumericStrings(value: unknown): unknown {
  return coerceStringsDeep(value, [false]);
}

function coerceStringsDeep(value: unknown, mutated: [boolean]): unknown {
  if (typeof value === "string") return coerceNumericString(value, mutated);
  if (Array.isArray(value)) return coerceStringsInArray(value, mutated);
  if (isPlainObject(value)) return coerceStringsInObject(value, mutated);
  return value;
}

function coerceNumericString(value: string, mutated: [boolean]): string | number | bigint {
  const trimmed = value.trim();
  if (trimmed === "") return value;
  if (/^-?\d+$/.test(trimmed)) {
    const asNumber = Number(trimmed);
    if (Number.isSafeInteger(asNumber)) {
      mutated[0] = true;
      return BigInt(asNumber);
    }
    return value;
  }
  const decimal = Number(trimmed);
  if (Number.isFinite(decimal)) {
    mutated[0] = true;
    return decimal;
  }
  return value;
}

function coerceStringsInArray(value: unknown[], mutated: [boolean]): unknown[] {
  const child: [boolean] = [false];
  const out = value.map((item) => coerceStringsDeep(item, child));
  if (!child[0]) return value;
  mutated[0] = true;
  return out;
}

function coerceStringsInObject(value: Record<string, unknown>, mutated: [boolean]): Record<string, unknown> {
  const child: [boolean] = [false];
  const out: Record<string, unknown> = {};
  for (const [key, val] of Object.entries(value)) {
    out[key] = coerceStringsDeep(val, child);
  }
  if (!child[0]) return value;
  mutated[0] = true;
  return out;
}

/**
 * Convert safe-integer `BigInt` values in the eval result back to plain JS
 * `number` so downstream formatters (`toFiniteNumber`, `JSON.stringify` on
 * row keys, chart aggregators) continue to work. BigInts outside safe
 * integer range flow through as `Number(...)` — dashboards use these
 * primarily for time deltas and counts, both of which stay comfortably
 * inside `Number.MAX_SAFE_INTEGER`.
 */
function normalizeCelValue(value: unknown): unknown {
  if (typeof value === "bigint") {
    return Number(value);
  }
  if (Array.isArray(value)) {
    return value.map(normalizeCelValue);
  }
  if (isPlainObject(value)) {
    const out: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(value)) {
      out[key] = normalizeCelValue(val);
    }
    return out;
  }
  return value;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (!value || typeof value !== "object") return false;
  if (Array.isArray(value)) return false;
  if (value instanceof Date) return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

/**
 * Shared CEL environment. Instantiated once because `new Environment(...)`
 * plus function registration is measurably expensive; the library docs
 * flag it as a hot-path concern.
 */
const ENV = new Environment({
  unlistedVariablesAreDyn: true,
  homogeneousAggregateLiterals: false,
});

registerCustomFunctions(ENV);
