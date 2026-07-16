// Custom function registrations layered on top of `@marcbachmann/cel-js`
// for the dashboard's expression environment. Split out of `celExpr.ts` so
// the adapter surface stays focused on parse / eval / coercion.
import type { Environment } from "@marcbachmann/cel-js";

import { coerceWidgetTimestamp } from "./widgetFormat";

export function registerCustomFunctions(env: Environment): void {
  registerCoercionHelpers(env);
  registerStringPredicates(env);
  registerCaseHelpers(env);
  registerDateHelpers(env);
  registerJsonAndList(env);
  registerStringHelpers(env);
  registerAvatarHelpers(env);
}

function registerCoercionHelpers(env: Environment): void {
  // `float()` — the library ships an int/uint overload but not a string or
  // dyn conversion, so we fill the gap. Fail-soft: any unparseable value
  // becomes 0, matching the legacy behavior authors relied on inside
  // templates like `{{ float(value) * 100 }}`.
  env.registerFunction("float(dyn): double", (value: unknown) => toFloat(value));

  // Library `int(string)` throws on unparseable / whitespace-padded input and
  // cannot be overridden (signature overlap). `compileExpr` rewrites
  // `int(...)` calls to `__dashboardInt(...)` so this fail-soft dyn
  // handler restores the legacy `0` contract for templates like
  // `{{ int(value) / 2 }}`.
  env.registerFunction("__dashboardInt(dyn): int", (value: unknown) => toInt(value));

  // Library `string(...)` only covers scalars. Maps and lists used to
  // JSON-serialize via the dashboard helper; restore that for
  // `{{ string(payload) }}` cells.
  env.registerFunction("string(map): string", (value: unknown) => stringifyCelValue(value));
  env.registerFunction("string(list): string", (value: unknown) => stringifyCelValue(value));
}

function registerStringPredicates(env: Environment): void {
  // String predicates in function form. `@marcbachmann/cel-js` exposes
  // these as string methods (`s.contains(x)`) but existing dashboards
  // (and the row `where` filters) use the function form.
  env.registerFunction(
    "contains(dyn, dyn): bool",
    (s: unknown, sub: unknown) => typeof s === "string" && typeof sub === "string" && s.includes(sub),
  );
  env.registerFunction(
    "startsWith(dyn, dyn): bool",
    (s: unknown, p: unknown) => typeof s === "string" && typeof p === "string" && s.startsWith(p),
  );
  env.registerFunction(
    "endsWith(dyn, dyn): bool",
    (s: unknown, p: unknown) => typeof s === "string" && typeof p === "string" && s.endsWith(p),
  );
  env.registerFunction("matches(dyn, dyn): bool", (s: unknown, re: unknown) => {
    if (typeof s !== "string" || typeof re !== "string") return false;
    try {
      return new RegExp(re).test(s);
    } catch {
      return false;
    }
  });
}

function registerCaseHelpers(env: Environment): void {
  env.registerFunction("lower(dyn): string", (s: unknown) => (s == null ? "" : String(s).toLowerCase()));
  env.registerFunction("upper(dyn): string", (s: unknown) => (s == null ? "" : String(s).toUpperCase()));
}

function registerDateHelpers(env: Environment): void {
  // `duration()` — the library's builtin `duration(string)` returns a
  // `google.protobuf.Duration`. We register the int/double overloads to
  // provide the "seconds → human string" formatter authors reach for.
  // Protobuf Duration results are normalized to the same human string in
  // `normalizeCelValue` so `{{ duration("5m") }}` still renders `5m`.
  env.registerFunction("duration(int): string", (seconds: unknown) => formatDurationSeconds(toNumber(seconds)));
  env.registerFunction("duration(double): string", (seconds: unknown) => formatDurationSeconds(toNumber(seconds)));

  env.registerFunction("formatDate(dyn, string): string", (value: unknown, pattern: unknown) =>
    formatDate(value, pattern),
  );

  // `epochMs` returns `int` (BigInt) so authors can subtract two epoch
  // timestamps and divide by 1000 without accidentally crossing into
  // double arithmetic. Fail-soft: unparseable input yields 0.
  env.registerFunction("epochMs(dyn): int", (value: unknown) => {
    const date = coerceWidgetTimestamp(unwrapNumeric(value));
    return date ? BigInt(date.getTime()) : 0n;
  });
}

/** Exported for the eval adapter: format a protobuf-style Duration as `5m`. */
export function formatDurationSeconds(value: number): string {
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

function registerJsonAndList(env: Environment): void {
  env.registerFunction("parseJson(dyn): dyn", (value: unknown) => parseJsonValue(value));

  // `join` is exposed as a builtin so authors can flatten the result of a
  // `.map` macro chain into a single string. Fail-soft: anything that
  // isn't a list returns `""`, missing / non-string separators collapse to
  // `""`, and null/undefined elements render as `""` so a careless map
  // doesn't smear `null` strings into the output.
  env.registerFunction("join(dyn, dyn): string", (list: unknown, sep: unknown) => joinList(list, sep));
}

function registerStringHelpers(env: Environment): void {
  // String-trimming helpers registered with `dyn` first args so authors
  // can pass numeric values (e.g. `substring(pr_number, 0, 3)`) without
  // manually calling `string()`. `null` / `undefined` collapses to `""`
  // to match the legacy fail-soft contract.
  env.registerFunction("substring(dyn, dyn, dyn): string", (s: unknown, start: unknown, end: unknown) =>
    substringOf(s, start, end),
  );
  env.registerFunction("substring(dyn, dyn): string", (s: unknown, start: unknown) => substringOf(s, start, undefined));
  env.registerFunction("truncate(dyn, dyn, dyn): string", (s: unknown, n: unknown, suffix: unknown) =>
    truncateStr(s, n, suffix),
  );
  env.registerFunction("truncate(dyn, dyn): string", (s: unknown, n: unknown) => truncateStr(s, n, undefined));
  env.registerFunction("firstLine(dyn): string", (s: unknown) => firstLineOf(s));
  env.registerFunction("splitIndex(dyn, dyn, dyn): string", (s: unknown, sep: unknown, i: unknown) =>
    splitIndex(s, sep, i),
  );
  env.registerFunction("trim(dyn): string", (s: unknown) => trimStr(s, undefined));
  env.registerFunction("trim(dyn, dyn): string", (s: unknown, chars: unknown) => trimStr(s, chars));
  env.registerFunction("replace(dyn, dyn, dyn): string", (s: unknown, oldStr: unknown, newStr: unknown) =>
    replaceStr(s, oldStr, newStr),
  );
  env.registerFunction("indexOf(dyn, dyn): int", (s: unknown, sub: unknown) => BigInt(indexOfStr(s, sub)));
}

function registerAvatarHelpers(env: Environment): void {
  // `firstInitial` and `githubAvatarOrInitial` need explicit overloads
  // for each arity because CEL doesn't support optional / variadic
  // function parameters.
  env.registerFunction("initial(dyn): string", (value: unknown) => initialLetter(value));
  env.registerFunction("firstInitial(dyn): string", (a: unknown) => firstInitialFromValues(a));
  env.registerFunction("firstInitial(dyn, dyn): string", (a: unknown, b: unknown) => firstInitialFromValues(a, b));
  env.registerFunction("firstInitial(dyn, dyn, dyn): string", (a: unknown, b: unknown, c: unknown) =>
    firstInitialFromValues(a, b, c),
  );
  env.registerFunction("firstInitial(dyn, dyn, dyn, dyn): string", (a: unknown, b: unknown, c: unknown, d: unknown) =>
    firstInitialFromValues(a, b, c, d),
  );
  env.registerFunction("githubAvatarOrInitial(dyn): string", (author: unknown) => githubAvatar(author, undefined));
  env.registerFunction("githubAvatarOrInitial(dyn, dyn): string", (author: unknown, committer: unknown) =>
    githubAvatar(author, committer),
  );
}

function toNumber(value: unknown): number {
  if (typeof value === "number") return value;
  if (typeof value === "bigint") return Number(value);
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string") {
    const n = Number(value);
    return Number.isFinite(n) ? n : NaN;
  }
  return NaN;
}

function unwrapNumeric(value: unknown): unknown {
  return typeof value === "bigint" ? Number(value) : value;
}

function toFloat(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "bigint") return Number(value);
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string") {
    const n = Number(value);
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
}

function toInt(value: unknown): bigint {
  if (typeof value === "bigint") return value;
  if (typeof value === "number") return Number.isFinite(value) ? BigInt(Math.trunc(value)) : 0n;
  if (typeof value === "boolean") return value ? 1n : 0n;
  if (typeof value === "string") {
    const n = Number(value);
    return Number.isFinite(n) ? BigInt(Math.trunc(n)) : 0n;
  }
  return 0n;
}

export function stringifyCelValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (typeof value === "bigint") return value.toString();
  try {
    return JSON.stringify(value, (_key, nested) => {
      if (typeof nested !== "bigint") return nested;
      return Number.isSafeInteger(Number(nested)) ? Number(nested) : nested.toString();
    });
  } catch {
    return String(value);
  }
}

function coerceToString(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "bigint") return value.toString();
  return String(value);
}

// `parseJson` parses a JSON-encoded string into a structured CEL value.
// Already parsed non-string inputs pass through unchanged, and any parse
// failure returns `null` so downstream operations fail soft — matching
// the graceful-degrade convention used by `epochMs`, `matches`, and
// friends.
function parseJsonValue(value: unknown): unknown {
  if (value === null || value === undefined) return null;
  if (typeof value !== "string") return value;
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return null;
  }
}

function joinList(list: unknown, sep: unknown): string {
  if (!Array.isArray(list)) return "";
  const separator = typeof sep === "string" ? sep : "";
  return list.map((item) => stringifyJoinItem(item)).join(separator);
}

function stringifyJoinItem(item: unknown): string {
  if (item === null || item === undefined) return "";
  if (typeof item === "string") return item;
  if (typeof item === "bigint") return item.toString();
  return String(item);
}

function substringOf(s: unknown, start: unknown, end: unknown): string {
  if (s === null || s === undefined) return "";
  const text = coerceToString(s);
  if (text === "") return "";
  const startIndex = clampIndex(start, text.length);
  if (end === undefined) return text.slice(startIndex);
  const endIndex = clampIndex(end, text.length);
  if (endIndex <= startIndex) return "";
  return text.slice(startIndex, endIndex);
}

function truncateStr(s: unknown, n: unknown, suffix: unknown): string {
  const text = coerceToString(s);
  const limit = toNumber(n);
  if (!Number.isFinite(limit) || limit < 0) return text;
  if (text.length <= limit) return text;
  const tail = typeof suffix === "string" ? suffix : "";
  return text.slice(0, Math.trunc(limit)) + tail;
}

function firstLineOf(s: unknown): string {
  const text = coerceToString(s);
  if (text === "") return "";
  const newline = text.search(/\r\n|\r|\n/);
  return newline === -1 ? text : text.slice(0, newline);
}

// Nth segment of `split(s, sep)`, returned as a scalar so authors don't
// run into the no-postfix-after-call-result limitation. Negative `i`
// counts from the end (`-1` = last). Out-of-range / non-numeric `i`
// returns `""`.
//
// When the separator is a bare newline we split on `/\r\n|\r|\n/` so that
// `\r\n` (Windows) and bare `\r` (classic Mac) line endings are treated
// the same as `\n`. This keeps `splitIndex(value, "\n", 0)` in agreement
// with `firstLine` — otherwise CRLF text would leave a trailing `\r` on
// segments. `@marcbachmann/cel-js` already interprets `"\n"` in CEL
// source as a real newline, so we no longer need to unescape the
// separator ourselves.
function splitIndex(s: unknown, sep: unknown, i: unknown): string {
  const text = coerceToString(s);
  const separator = typeof sep === "string" ? sep : String(sep ?? "");
  if (separator === "") return text;
  const parts = separator === "\n" ? text.split(/\r\n|\r|\n/) : text.split(separator);
  const raw = toNumber(i);
  if (!Number.isFinite(raw)) return "";
  const index = raw < 0 ? parts.length + Math.trunc(raw) : Math.trunc(raw);
  if (index < 0 || index >= parts.length) return "";
  return parts[index];
}

function trimStr(s: unknown, chars: unknown): string {
  const text = coerceToString(s);
  if (chars === undefined) return text.trim();
  const charset = coerceToString(chars);
  if (charset === "") return text;
  let start = 0;
  let end = text.length;
  while (start < end && charset.includes(text[start])) start++;
  while (end > start && charset.includes(text[end - 1])) end--;
  return text.slice(start, end);
}

function replaceStr(s: unknown, oldStr: unknown, newStr: unknown): string {
  const text = coerceToString(s);
  const search = coerceToString(oldStr);
  if (search === "") return text;
  const replacement = coerceToString(newStr);
  return text.split(search).join(replacement);
}

function indexOfStr(s: unknown, sub: unknown): number {
  const text = coerceToString(s);
  const needle = coerceToString(sub);
  return text.indexOf(needle);
}

function initialLetter(value: unknown): string {
  const text = coerceToString(value).trim();
  if (text === "") return "";
  const match = text.match(/[A-Za-z0-9]/);
  return match ? match[0].toUpperCase() : text.charAt(0).toUpperCase();
}

function firstInitialFromValues(...values: unknown[]): string {
  for (const candidate of values) {
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

// Renders a deployer avatar for GitHub webhook author/committer maps.
// Uses the GitHub avatar image when `author.username` is present;
// otherwise falls back to an initial-letter badge derived from the
// available names.
function githubAvatar(author: unknown, committer: unknown): string {
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
}

function clampIndex(value: unknown, length: number): number {
  const raw = toNumber(value);
  if (!Number.isFinite(raw)) return 0;
  const truncated = Math.trunc(raw);
  if (truncated < 0) return Math.max(0, length + truncated);
  if (truncated > length) return length;
  return truncated;
}

/**
 * Format a date value using a small token pattern (e.g. `MM/dd`, `yyyy-MM-dd HH:mm`).
 *
 * The value may be an ISO-8601 string, a `Date` instance, an epoch number in
 * seconds (`< 1e11`), or an epoch number in milliseconds (`>= 1e11`). All
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
  const date = coerceWidgetTimestamp(unwrapNumeric(value));
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
