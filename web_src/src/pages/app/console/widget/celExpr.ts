// CEL expression support for dashboard table panels.
//
// Strings wrapped in `{{ ... }}` are compiled once and evaluated per row.
// Strings without braces use legacy dot-path semantics via callers.
import { type CstNode } from "chevrotain";
import { CelTypeError, evaluate as celEvaluate, parse as celParse } from "cel-js";

import { coerceWidgetTimestamp } from "./widgetFormat";

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

/**
 * Identifier substituted for bare `$` so cel-js (which only accepts
 * `[a-zA-Z_][a-zA-Z0-9_]*` identifiers) can parse canvas-style node refs
 * like `$["deploy-prod"].outputs.url`. Rows that need to resolve those
 * references must carry the same map under this key.
 */
export const DOLLAR_REWRITE_IDENTIFIER = "__runNodes__";

/**
 * Rewrite bare `$` tokens to `DOLLAR_REWRITE_IDENTIFIER` outside of string
 * literals so cel-js can parse expressions like `$["node name"].outputs.x`.
 * Single- and double-quoted strings are copied verbatim (with backslash
 * escapes preserved) so a `$` inside a string literal is left alone.
 *
 * The rewrite is purely additive: cel-js's lexer rejects bare `$` today,
 * so no existing expression depended on the previous behavior.
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
  const result = celParse(rewritten);
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
  formatDate,
  // Convert any date-like value (ISO string, Date, epoch number) to
  // ms-since-epoch. Returns 0 for unparseable input so arithmetic doesn't
  // blow up — authors checking `epochMs(x) > 0` can still detect missing
  // values. Pairs with `duration()` for human-friendly output, e.g.
  // `{{ duration((epochMs(finishedAt) - epochMs(createdAt)) / 1000) }}`.
  epochMs: (value: unknown): number => {
    const date = coerceWidgetTimestamp(value);
    return date ? date.getTime() : 0;
  },
  // Parse a JSON-encoded string into a structured CEL value (list, map, or
  // scalar) so authors can `.map`, `.filter`, or dot-access the result. Already
  // parsed non-string inputs pass through unchanged, and any parse failure
  // returns `null` so downstream operations fail soft — matching the
  // graceful-degrade convention used by `epochMs`, `matches`, and friends.
  parseJson: (value: unknown): unknown => {
    if (value === null || value === undefined) return null;
    if (typeof value !== "string") return value;
    try {
      return JSON.parse(value) as unknown;
    } catch {
      return null;
    }
  },
  // Join the elements of a list with a separator. Exposed as a builtin (not a
  // method) because cel-js's parser refuses `.foo()` chains after a
  // function-call result — so `list.map(x, expr).join("")` would not parse.
  // Authors instead write `join(list.map(x, expr), "")`. Fail-soft: anything
  // that isn't a list returns the empty string, missing/non-string separators
  // collapse to "", and null/undefined elements become "" so a careless map
  // doesn't smear `null` strings into the output.
  join: (list: unknown, sep: unknown): string => {
    if (!Array.isArray(list)) return "";
    const separator = typeof sep === "string" ? sep : "";
    return list.map((item) => stringifyJoinItem(item)).join(separator);
  },
  // String-trimming helpers below all return scalars directly. They exist
  // because cel-js doesn't allow postfix `[i]` / `.method()` after a
  // function-call result, so an author can't write `split(s, "\n")[0]` or
  // `s.substring(0, 80)` — they have to call a single helper that already
  // returns the scalar they want. This matches the expr-lang surface used at
  // node-config / write time (see `web_src/src/lib/exprEvaluator.ts`).
  // Fail-soft: non-string `s` coerces via `String(s)` (matching `lower`/
  // `upper`); `null` / `undefined` input returns `""`.
  substring: (s: unknown, start: unknown, end?: unknown): string => {
    const text = coerceToString(s);
    if (text === "") return "";
    const startIndex = clampIndex(start, text.length);
    if (end === undefined) return text.slice(startIndex);
    const endIndex = clampIndex(end, text.length);
    if (endIndex <= startIndex) return "";
    return text.slice(startIndex, endIndex);
  },
  // First `n` characters of `s`, with an optional suffix (e.g. "…") appended
  // only when truncation actually happened. Authors reach for this to keep
  // long run outputs from blowing up table cells while still hinting that the
  // value was clipped.
  truncate: (s: unknown, n: unknown, suffix?: unknown): string => {
    const text = coerceToString(s);
    const limit = Number(n);
    if (!Number.isFinite(limit) || limit < 0) return text;
    if (text.length <= limit) return text;
    const tail = typeof suffix === "string" ? suffix : "";
    return text.slice(0, Math.trunc(limit)) + tail;
  },
  // Text before the first newline. Treats `\r\n` and bare `\r` the same as
  // `\n` so Windows / classic-Mac line endings don't sneak through. Empty
  // input stays empty.
  firstLine: (s: unknown): string => {
    const text = coerceToString(s);
    if (text === "") return "";
    const newline = text.search(/\r\n|\r|\n/);
    return newline === -1 ? text : text.slice(0, newline);
  },
  // Nth segment of `split(s, sep)`, returned as a scalar so authors don't run
  // into the no-postfix-after-call-result limitation. Negative `i` counts
  // from the end (`-1` = last). Out-of-range / non-numeric `i` returns "".
  //
  // The separator is run through `unescapeSeparator` because cel-js does
  // **not** interpret backslash escapes in string literals — `"\n"` written
  // in a CEL expression is the literal two-character string `\n`. Without
  // unescaping, the natural `splitIndex(message, "\n", 0)` form would never
  // match an actual newline. We translate `\n`, `\r`, `\t`, and `\\` so the
  // common cases just work.
  //
  // When the separator is a bare newline we split on `/\r\n|\r|\n/` so that
  // `\r\n` (Windows) and bare `\r` (classic Mac) line endings are treated the
  // same as `\n`. This keeps `splitIndex(value, "\n", 0)` in agreement with
  // `firstLine` — otherwise CRLF text would leave a trailing `\r` on segments.
  splitIndex: (s: unknown, sep: unknown, i: unknown): string => {
    const text = coerceToString(s);
    const separator = unescapeSeparator(typeof sep === "string" ? sep : String(sep ?? ""));
    if (separator === "") return text;
    const parts = separator === "\n" ? text.split(/\r\n|\r|\n/) : text.split(separator);
    const raw = Number(i);
    if (!Number.isFinite(raw)) return "";
    const index = raw < 0 ? parts.length + Math.trunc(raw) : Math.trunc(raw);
    if (index < 0 || index >= parts.length) return "";
    return parts[index];
  },
  // Rounds out string parity with expr-lang. `trim` removes leading /
  // trailing whitespace (or, when `chars` is supplied, leading / trailing
  // characters from `chars`). `replace` swaps every `old` with `new`.
  // `indexOf` returns -1 when missing, matching JS / expr-lang behavior.
  trim: (s: unknown, chars?: unknown): string => {
    const text = coerceToString(s);
    if (chars === undefined) return text.trim();
    const charset = coerceToString(chars);
    if (charset === "") return text;
    let start = 0;
    let end = text.length;
    while (start < end && charset.includes(text[start])) start++;
    while (end > start && charset.includes(text[end - 1])) end--;
    return text.slice(start, end);
  },
  replace: (s: unknown, oldStr: unknown, newStr: unknown): string => {
    const text = coerceToString(s);
    const search = coerceToString(oldStr);
    if (search === "") return text;
    const replacement = coerceToString(newStr);
    return text.split(search).join(replacement);
  },
  indexOf: (s: unknown, sub: unknown): number => {
    const text = coerceToString(s);
    const needle = coerceToString(sub);
    return text.indexOf(needle);
  },
  // First letter of a display name, uppercased. Skips leading whitespace and
  // prefers the first alphanumeric character so values like "cloud-robot"
  // render as "C". Returns "" for missing / empty input.
  initial: (value: unknown): string => initialLetter(value),
  // Walks the provided values in order and returns the first non-empty
  // initial. Authors use this for avatar fallbacks when a GitHub username is
  // unavailable but a human/bot display name is still present.
  firstInitial: (a: unknown, b?: unknown, c?: unknown, d?: unknown): string => firstInitialFromValues(a, b, c, d),
  // Renders a deployer avatar for GitHub webhook author/committer maps.
  // Uses the GitHub avatar image when `author.username` is present; otherwise
  // falls back to an initial-letter badge derived from the available names.
  githubAvatarOrInitial: (author: unknown, committer?: unknown): string => {
    const authorRecord = asRecord(author);
    const committerRecord = asRecord(committer);
    const username = coerceToString(authorRecord?.username).trim();
    if (username) {
      return `<img class="avatar avatar-image" src="https://github.com/${username}.png" alt="" />`;
    }
    const letter = firstInitialFromValues(
      authorRecord?.name,
      committerRecord?.name,
      authorRecord?.username,
      committerRecord?.username,
    );
    if (!letter) return "";
    return `<div class="avatar avatar-fallback">${letter}</div>`;
  },
};

function coerceToString(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  return String(value);
}

function initialLetter(value: unknown): string {
  const text = coerceToString(value).trim();
  if (text === "") return "";
  const match = text.match(/[A-Za-z0-9]/);
  return match ? match[0].toUpperCase() : text.charAt(0).toUpperCase();
}

function firstInitialFromValues(a: unknown, b?: unknown, c?: unknown, d?: unknown): string {
  for (const candidate of [a, b, c, d]) {
    if (candidate === undefined) continue;
    const letter = initialLetter(candidate);
    if (letter) return letter;
  }
  return "";
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function clampIndex(value: unknown, length: number): number {
  const raw = Number(value);
  if (!Number.isFinite(raw)) return 0;
  const truncated = Math.trunc(raw);
  if (truncated < 0) return Math.max(0, length + truncated);
  if (truncated > length) return length;
  return truncated;
}

/**
 * Translate the common backslash escapes (`\n`, `\r`, `\t`, `\\`) in a
 * separator string into their literal characters. cel-js's lexer copies
 * string literals verbatim — it does not honor escape sequences — so a
 * separator authored as `"\n"` arrives here as two characters. Anything
 * other than the recognized pair is preserved as-is so a literal `\` in
 * (for example) a Windows path separator still works.
 */
function unescapeSeparator(raw: string): string {
  if (!raw.includes("\\")) return raw;
  let out = "";
  for (let i = 0; i < raw.length; i++) {
    const ch = raw[i];
    if (ch !== "\\" || i + 1 >= raw.length) {
      out += ch;
      continue;
    }
    const next = raw[i + 1];
    if (next === "n") {
      out += "\n";
      i++;
    } else if (next === "r") {
      out += "\r";
      i++;
    } else if (next === "t") {
      out += "\t";
      i++;
    } else if (next === "\\") {
      out += "\\";
      i++;
    } else {
      out += ch;
    }
  }
  return out;
}

function stringifyJoinItem(item: unknown): string {
  if (item === null || item === undefined) return "";
  if (typeof item === "string") return item;
  return String(item);
}

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

/**
 * Format a date value using a small token pattern (e.g. `MM/dd`, `yyyy-MM-dd HH:mm`).
 *
 * The value may be an ISO-8601 string, a `Date` instance, an epoch number in
 * seconds (`< 1e12`), or an epoch number in milliseconds (`>= 1e12`). All
 * tokens render in the viewer's local time, matching the rest of the
 * dashboard's display conventions (`widgetFormat.ts` also uses `toLocale*`).
 *
 * Returns an empty string when the value cannot be parsed or when the pattern
 * is missing — this is consistent with the other builtins and keeps widgets
 * resilient to malformed source data.
 *
 * Supported tokens (longest matched first):
 *   yyyy, yy        – four / two digit year
 *   MM, M           – two / one-or-two digit month (1-12)
 *   dd, d           – two / one-or-two digit day of month
 *   HH, H           – two / one-or-two digit hour (0-23)
 *   mm, m           – two / one-or-two digit minute
 *   ss, s           – two / one-or-two digit second
 *
 * Other characters in the pattern are preserved literally. Authors who need a
 * literal letter that overlaps a token should pick a different separator.
 */
function formatDate(value: unknown, pattern: unknown): string {
  if (typeof pattern !== "string" || pattern === "") return "";
  const date = coerceWidgetTimestamp(value);
  if (!date) return "";
  return formatDateTokens(date, pattern);
}

const DATE_TOKEN_RE = /yyyy|yy|MM|M|dd|d|HH|H|mm|m|ss|s/g;

function formatDateTokens(date: Date, pattern: string): string {
  return pattern.replace(DATE_TOKEN_RE, (token) => {
    switch (token) {
      case "yyyy":
        return String(date.getFullYear());
      case "yy":
        return String(date.getFullYear() % 100).padStart(2, "0");
      case "MM":
        return String(date.getMonth() + 1).padStart(2, "0");
      case "M":
        return String(date.getMonth() + 1);
      case "dd":
        return String(date.getDate()).padStart(2, "0");
      case "d":
        return String(date.getDate());
      case "HH":
        return String(date.getHours()).padStart(2, "0");
      case "H":
        return String(date.getHours());
      case "mm":
        return String(date.getMinutes()).padStart(2, "0");
      case "m":
        return String(date.getMinutes());
      case "ss":
        return String(date.getSeconds()).padStart(2, "0");
      case "s":
        return String(date.getSeconds());
      default:
        return token;
    }
  });
}

export type EvalResult = { ok: true; value: unknown } | { ok: false; error: string };

/**
 * Evaluate a compiled CEL expression. Returns `undefined` on any error so
 * existing callers (column rendering, payload merging, etc.) keep failing
 * gracefully. Numeric-looking string fields are silently retried as numbers
 * when a `CelTypeError` is raised — memory rows commonly stringify scalars,
 * and authors expect `{{ value / 2 }}` to "just work" on those rows.
 */
export function evalExpr(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): unknown {
  const detailed = evalExprDetailed(compiled, row, env);
  return detailed.ok ? detailed.value : undefined;
}

/**
 * Same as `evalExpr` but surfaces compile/eval errors so the editor can
 * display them in inline previews. The error string is the first compile
 * error or the message from the underlying eval error.
 */
export function evalExprDetailed(compiled: CompiledExpr, row: Record<string, unknown>, env: ExprEnv): EvalResult {
  if (!compiled.ok) return { ok: false, error: compiled.error };
  const vars = { ...env.globals, ...row };
  try {
    return { ok: true, value: celEvaluate(compiled.cst, vars, env.functions) };
  } catch (initial) {
    if (!(initial instanceof CelTypeError)) {
      return { ok: false, error: initial instanceof Error ? initial.message : String(initial) };
    }
    const coerced = coerceNumericStrings(vars);
    if (coerced === vars) {
      return { ok: false, error: initial.message };
    }
    try {
      return { ok: true, value: celEvaluate(compiled.cst, coerced, env.functions) };
    } catch (retry) {
      const message = retry instanceof Error ? retry.message : String(retry);
      return { ok: false, error: message };
    }
  }
}

/**
 * Returns a new vars object with string values that parse cleanly as finite
 * numbers replaced with their numeric form. Non-numeric strings are left
 * alone so equality and substring checks still work. When nothing changes
 * the original object is returned so callers can short-circuit.
 */
function coerceNumericStrings(vars: Record<string, unknown>): Record<string, unknown> {
  let changed = false;
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(vars)) {
    if (typeof value === "string" && value.trim() !== "") {
      const n = Number(value);
      if (Number.isFinite(n)) {
        out[key] = n;
        changed = true;
        continue;
      }
    }
    out[key] = value;
  }
  return changed ? out : vars;
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
