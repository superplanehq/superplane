/**
 * expr-suggester.ts
 *
 * Headless suggestion engine for Expr-style expressions where:
 * - `$` is the ONLY environment root (no $env).
 *
 * Supports:
 * 1) Global variables (dynamic) you pass in at runtime via `globals`
 *    - `$["key"]` resolves to globals["key"]
 *    - `$["key"].nested` resolves to nested properties
 * 2) All built-in functions (listed below) with snippet insertText
 * 3) Env key completions:
 *    - after "$" (e.g. take($) / take($ -> suggests keys as $["key"] or $["key"].)
 *    - after "$[" (e.g. $[ -> suggests keys)
 *    - inside $["... (prefix filtering)
 * 4) Dot completions for object/map fields:
 *    - user.profile.
 *    - $["key"].
 *    - abs($["test"].  (works inside function calls by extracting tail path)
 *
 * Enhancement:
 * - When inserting a VARIABLE suggestion (global name or $["key"]), if the value is expandable
 *   (object/array), we append a trailing '.' so the user can keep typing members.
 */

export type SuggestionKind = "function" | "variable" | "field" | "keyword";

export interface Suggestion {
  label: string;
  kind: SuggestionKind;
  insertText?: string;
  detail?: string;
  labelDetail?: string;
}

export interface GetSuggestionsOptions {
  includeFunctions?: boolean;
  includeGlobals?: boolean;
  limit?: number;
  allowInStrings?: boolean;
}

type ExprFunction = { name: string; snippet?: string };

/** Built-in functions (from expr-lang docs categories). */
export const EXPR_FUNCTIONS: readonly ExprFunction[] = [
  // String
  { name: "trim", snippet: "trim(${1:str}${2:, ${3:chars}})" },
  { name: "trimPrefix", snippet: "trimPrefix(${1:str}, ${2:prefix})" },
  { name: "trimSuffix", snippet: "trimSuffix(${1:str}, ${2:suffix})" },
  { name: "upper", snippet: "upper(${1:str})" },
  { name: "lower", snippet: "lower(${1:str})" },
  { name: "split", snippet: "split(${1:str}, ${2:delimiter}${3:, ${4:n}})" },
  { name: "splitAfter", snippet: "splitAfter(${1:str}, ${2:delimiter}${3:, ${4:n}})" },
  { name: "replace", snippet: "replace(${1:str}, ${2:old}, ${3:new})" },
  { name: "repeat", snippet: "repeat(${1:str}, ${2:n})" },
  { name: "indexOf", snippet: "indexOf(${1:str}, ${2:substring})" },
  { name: "lastIndexOf", snippet: "lastIndexOf(${1:str}, ${2:substring})" },
  { name: "hasPrefix", snippet: "hasPrefix(${1:str}, ${2:prefix})" },
  { name: "hasSuffix", snippet: "hasSuffix(${1:str}, ${2:suffix})" },

  // Date
  { name: "now", snippet: "now()" },
  { name: "duration", snippet: "duration(${1:str})" },
  { name: "date", snippet: "date(${1:str}${2:, ${3:format}}${4:, ${5:timezone}})" },
  { name: "timezone", snippet: "timezone(${1:str})" },

  // Number
  { name: "max", snippet: "max(${1:n1}, ${2:n2})" },
  { name: "min", snippet: "min(${1:n1}, ${2:n2})" },
  { name: "abs", snippet: "abs(${1:n})" },
  { name: "ceil", snippet: "ceil(${1:n})" },
  { name: "floor", snippet: "floor(${1:n})" },
  { name: "round", snippet: "round(${1:n})" },

  // Array
  { name: "all", snippet: "all(${1:array}, ${2:predicate})" },
  { name: "any", snippet: "any(${1:array}, ${2:predicate})" },
  { name: "one", snippet: "one(${1:array}, ${2:predicate})" },
  { name: "none", snippet: "none(${1:array}, ${2:predicate})" },
  { name: "map", snippet: "map(${1:array}, ${2:predicate})" },
  { name: "filter", snippet: "filter(${1:array}, ${2:predicate})" },
  { name: "find", snippet: "find(${1:array}, ${2:predicate})" },
  { name: "findIndex", snippet: "findIndex(${1:array}, ${2:predicate})" },
  { name: "findLast", snippet: "findLast(${1:array}, ${2:predicate})" },
  { name: "findLastIndex", snippet: "findLastIndex(${1:array}, ${2:predicate})" },
  { name: "groupBy", snippet: "groupBy(${1:array}, ${2:predicate})" },
  { name: "count", snippet: "count(${1:array}${2:, ${3:predicate}})" },
  { name: "concat", snippet: "concat(${1:array1}, ${2:array2}${3:, ${4:...}})" },
  { name: "flatten", snippet: "flatten(${1:array})" },
  { name: "uniq", snippet: "uniq(${1:array})" },
  { name: "join", snippet: "join(${1:array}${2:, ${3:delimiter}})" },
  { name: "reduce", snippet: "reduce(${1:array}, ${2:predicate}${3:, ${4:initialValue}})" },
  { name: "sum", snippet: "sum(${1:array}${2:, ${3:predicate}})" },
  { name: "mean", snippet: "mean(${1:array})" },
  { name: "median", snippet: "median(${1:array})" },
  { name: "first", snippet: "first(${1:array})" },
  { name: "last", snippet: "last(${1:array})" },
  { name: "take", snippet: "take(${1:array}, ${2:n})" },
  { name: "reverse", snippet: "reverse(${1:array})" },
  { name: "sort", snippet: "sort(${1:array}${2:, ${3:order}})" },
  { name: "sortBy", snippet: "sortBy(${1:array}${2:, ${3:predicate}}${4:, ${5:order}})" },

  // Map
  { name: "keys", snippet: "keys(${1:map})" },
  { name: "values", snippet: "values(${1:map})" },

  // Type conversion
  { name: "type", snippet: "type(${1:v})" },
  { name: "int", snippet: "int(${1:v})" },
  { name: "float", snippet: "float(${1:v})" },
  { name: "string", snippet: "string(${1:v})" },
  { name: "toJSON", snippet: "toJSON(${1:v})" },
  { name: "fromJSON", snippet: "fromJSON(${1:v})" },
  { name: "toBase64", snippet: "toBase64(${1:v})" },
  { name: "fromBase64", snippet: "fromBase64(${1:v})" },
  { name: "toPairs", snippet: "toPairs(${1:map})" },
  { name: "fromPairs", snippet: "fromPairs(${1:array})" },

  // Misc
  { name: "len", snippet: "len(${1:v})" },
  { name: "get", snippet: "get(${1:v}, ${2:index})" },

  // Bitwise
  { name: "bitand", snippet: "bitand(${1:a}, ${2:b})" },
  { name: "bitor", snippet: "bitor(${1:a}, ${2:b})" },
  { name: "bitxor", snippet: "bitxor(${1:a}, ${2:b})" },
  { name: "bitnand", snippet: "bitnand(${1:a}, ${2:b})" },
  { name: "bitnot", snippet: "bitnot(${1:a})" },
  { name: "bitshl", snippet: "bitshl(${1:a}, ${2:b})" },
  { name: "bitshr", snippet: "bitshr(${1:a}, ${2:b})" },
  { name: "bitushr", snippet: "bitushr(${1:a}, ${2:b})" },
] as const;

export function getSuggestions<TGlobals extends Record<string, unknown>>(
  text: string,
  cursor: number,
  globals: TGlobals,
  options: GetSuggestionsOptions = {},
): Suggestion[] {
  const { includeFunctions = true, includeGlobals = true, limit = 30, allowInStrings = false } = options;
  const left = text.slice(0, cursor);
  // 0) Env key trigger: after "$" or "$[" suggest keys immediately
  const envTrigger = detectEnvKeyTrigger(left);
  if (envTrigger) {
    const keys = listGlobalKeys(globals);
    return keys.slice(0, limit).map((k) => {
      const v = (globals as Record<string, unknown>)[k];
      const tailDot = isExpandableValue(v) ? "." : "";
      return {
        label: k,
        kind: "variable",
        insertText: `$[${quoteKey(k, envTrigger.quote)}]${tailDot}`,
        detail: getValueTypeLabel(v),
        labelDetail: formatNodeNameLabel(getNodeName(globals, k, v)),
      };
    });
  }

  // 1) Bracket key completion FIRST (because you're "inside a string" by definition)
  // NOTE: Here we are completing the KEY INSIDE $["... so we do NOT append a dot.
  const bracketCtx = detectBracketKeyContext(left);
  if (bracketCtx) {
    const prefix = (bracketCtx.partialKey ?? "").toLowerCase();
    const keys = listGlobalKeys(globals);
    return keys
      .filter((k) => k.toLowerCase().startsWith(prefix))
      .slice(0, limit)
      .map((k) => ({
        label: k,
        kind: "variable",
        insertText: quoteKey(k, bracketCtx.quote), // only the key token
        detail: getValueTypeLabel((globals as Record<string, unknown>)[k]),
        labelDetail: formatNodeNameLabel(getNodeName(globals, k, (globals as Record<string, unknown>)[k])),
      }));
  }

  // 2) Now it's safe to suppress suggestions inside normal strings
  if (!allowInStrings && isProbablyInsideString(left)) return [];

  // 3) Dot completion
  const dotCtx = detectDotContext(left);
  if (dotCtx) {
    const { baseExpr, memberPrefix, operator } = dotCtx;

    const resolvableBase = extractTailPathExpression(baseExpr);
    const target = resolveExprToValue(resolvableBase, globals);

    const keys = listKeys(target);

    const mp = (memberPrefix ?? "").toLowerCase();

    return keys
      .filter((k) => k.toLowerCase().startsWith(mp) && mp !== k.toLowerCase())
      .slice(0, limit)
      .map((k) => {
        const needsQuotes = needsQuotingAsIdentifier(k);

        // NEW: determine if THIS FIELD is expandable
        const fieldValue =
          target && (typeof target === "object" || typeof target === "function") ? getProp(target, k) : undefined;

        const tailDot = isExpandableValue(fieldValue) ? "." : "";

        if (operator === "?.") {
          return {
            label: k,
            kind: "field" as const,
            insertText: needsQuotes ? `?.["${escapeString(k)}"]${tailDot}` : `?.${k}${tailDot}`,
            detail: getValueTypeLabel(fieldValue),
          };
        }

        return {
          label: k,
          kind: "field" as const,
          insertText: needsQuotes ? `[${quoteKey(k, needsQuotes ? "'" : '"')}]${tailDot}` : `${k}${tailDot}`,
          detail: getValueTypeLabel(fieldValue),
        };
      });
  }

  // 4) Default completion (globals + functions)
  const prefix = getIdentifierPrefix(left).toLowerCase();
  const out: Suggestion[] = [];

  if (includeGlobals) {
    if (!prefix || "$".startsWith(prefix)) {
      out.push({
        label: "$",
        kind: "variable",
        insertText: "$",
        detail: getValueTypeLabel(globals),
      });
    }
  }

  if (includeFunctions) {
    for (const fn of EXPR_FUNCTIONS) {
      if (!prefix || fn.name.toLowerCase().startsWith(prefix)) {
        out.push({
          label: fn.name,
          kind: "function",
          insertText: fn.snippet ?? `${fn.name}($0)`,
          detail: "builtin function",
        });
      }
    }
  }

  out.sort((a, b) => rankSuggestion(a, b, prefix));
  return out.slice(0, limit);
}

/* ------------------------- Helpers ------------------------- */

type EnvKeyTrigger = { quote: "'" | '"' };

function detectEnvKeyTrigger(left: string): EnvKeyTrigger | null {
  if (/\$\s*$/.test(left)) return { quote: '"' };
  if (/\$\s*\[\s*$/.test(left)) return { quote: '"' };
  return null;
}

function listGlobalKeys(globals: Record<string, unknown>): string[] {
  const keys = Object.keys(globals ?? {});
  return keys.filter((key) => key !== "__nodeNames");
}

function isExpandableValue(v: unknown): boolean {
  if (v === null || typeof v !== "object") return false;
  return Object.values(v as Record<string, unknown>).length > 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function getNodeName(globals: Record<string, unknown>, key: string, value: unknown): string | undefined {
  if (isRecord(value)) {
    const nodeName = value.__nodeName;
    if (typeof nodeName === "string" && nodeName.trim()) return nodeName;
  }

  const nodeNames = isRecord(globals) ? (globals as Record<string, unknown>).__nodeNames : undefined;
  if (isRecord(nodeNames)) {
    const entry = nodeNames[key];
    if (typeof entry === "string" && entry.trim()) return entry;
    if (isRecord(entry)) {
      const name = entry.nodeName ?? entry.name ?? entry.label;
      if (typeof name === "string" && name.trim()) return name;
    }
  }

  return undefined;
}

function formatNodeNameLabel(nodeName?: string): string | undefined {
  if (!nodeName) return undefined;
  return `- (${nodeName})`;
}

function getValueTypeLabel(value: unknown): string {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  if (typeof value === "string") return "string";
  if (typeof value === "number") return "int";
  if (typeof value === "boolean") return "bool";
  if (typeof value === "object") return "object";
  return "unknown";
}

function rankSuggestion(a: Suggestion, b: Suggestion, prefixLower: string): number {
  const aLabel = a.label.toLowerCase();
  const bLabel = b.label.toLowerCase();
  const aStarts = prefixLower ? aLabel.startsWith(prefixLower) : true;
  const bStarts = prefixLower ? bLabel.startsWith(prefixLower) : true;

  if (aStarts !== bStarts) return aStarts ? -1 : 1;

  if (a.kind !== b.kind) {
    const order: Record<SuggestionKind, number> = {
      variable: 0,
      field: 1,
      function: 2,
      keyword: 3,
    };
    return (order[a.kind] ?? 99) - (order[b.kind] ?? 99);
  }

  if (a.label.length !== b.label.length) return a.label.length - b.label.length;
  return aLabel.localeCompare(bLabel);
}

function getIdentifierPrefix(left: string): string {
  const m = left.match(/[$A-Za-z_][$A-Za-z0-9_]*$/);
  return m ? m[0] : "";
}

type DotContext = { baseExpr: string; memberPrefix: string; operator: "." | "?." };

function detectDotContext(left: string): DotContext | null {
  if (left.endsWith(" ")) {
    return null;
  }

  const m = left.match(/(\$.+?)(\?\.|\.)\s*([$A-Za-z_][$A-Za-z0-9_]*)?$/);
  if (!m) return null;

  let baseExpr = (m[1] ?? "").trim();
  const operator = (m[2] === "?." ? "?." : ".") as "." | "?.";
  const memberPrefix = (m[3] ?? "").trim();
  if (!baseExpr) return null;

  if (baseExpr.includes("$")) {
    baseExpr = "$" + baseExpr.split("$").at(-1);
  }

  if (/^\d+(\.\d+)?$/.test(baseExpr)) return null;
  return { baseExpr, memberPrefix, operator };
}

type BracketKeyContext = { quote: "'" | '"'; partialKey: string };

function detectBracketKeyContext(left: string): BracketKeyContext | null {
  const m = left.match(/(?:\$\s*|\]\s*)\[\s*(['"])([^'"]*)$/);
  if (!m) return null;

  const quote = (m[1] === "'" ? "'" : '"') as "'" | '"';
  const partialKey = m[2] ?? "";
  return { quote, partialKey };
}

function extractTailPathExpression(expr: string): string {
  const s = expr.trim();
  let i = s.length - 1;

  let bracketDepth = 0;
  let inSingle = false;
  let inDouble = false;

  const isEscaped = (idx: number): boolean => {
    let bs = 0;
    for (let j = idx - 1; j >= 0 && s[j] === "\\"; j--) bs++;
    return bs % 2 === 1;
  };

  const isStopChar = (ch: string): boolean =>
    ch === "(" ||
    ch === ")" ||
    ch === "," ||
    ch === ";" ||
    ch === ":" ||
    ch === "?" ||
    ch === "+" ||
    ch === "-" ||
    ch === "*" ||
    ch === "/" ||
    ch === "%" ||
    ch === "|" ||
    ch === "&" ||
    ch === "!" ||
    ch === "=" ||
    ch === "<" ||
    ch === ">" ||
    ch === "\n" ||
    ch === "\r" ||
    ch === "\t" ||
    ch === " ";

  for (; i >= 0; i--) {
    const ch = s[i];

    if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
    else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

    if (inSingle || inDouble) continue;

    if (ch === "]") {
      bracketDepth++;
      continue;
    }
    if (ch === "[") {
      bracketDepth = Math.max(0, bracketDepth - 1);
      continue;
    }

    if (bracketDepth === 0 && isStopChar(ch)) {
      return s.slice(i + 1).trim();
    }
  }

  return s;
}

type Token = { t: "dot" } | { t: "ident"; v: string } | { t: "key"; v: string };

function resolveExprToValue<TGlobals extends Record<string, unknown>>(baseExpr: string, globals: TGlobals): unknown {
  const stripWhitespaceOutsideStrings = (input: string) => {
    let out = "";
    let inSingle = false;
    let inDouble = false;

    const isEscaped = (idx: number) => {
      let backslashes = 0;
      for (let j = idx - 1; j >= 0 && input[j] === "\\"; j--) backslashes++;
      return backslashes % 2 === 1;
    };

    for (let i = 0; i < input.length; i++) {
      const ch = input[i];
      if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
      else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

      if (!inSingle && !inDouble && /\s/u.test(ch)) {
        continue;
      }
      out += ch;
    }
    return out;
  };

  let expr = stripWhitespaceOutsideStrings(baseExpr.trim());

  const tokens: Token[] = [];
  let i = 0;
  const identRe = /^[$A-Za-z_][$A-Za-z0-9_]*/;

  while (i < expr.length) {
    const rest = expr.slice(i);

    if (rest[0] === ".") {
      tokens.push({ t: "dot" });
      i += 1;
      continue;
    }

    if (rest[0] === "[") {
      const quotedMatch = rest.match(/^\[\s*(['"])(.*?)\1\s*\]/);
      if (quotedMatch) {
        tokens.push({ t: "key", v: unescapeString(quotedMatch[2] ?? "") });
        i += quotedMatch[0].length;
        continue;
      }

      const numberMatch = rest.match(/^\[\s*(\d+)\s*\]/);
      if (numberMatch) {
        tokens.push({ t: "key", v: numberMatch[1] });
        i += numberMatch[0].length;
        continue;
      }

      return undefined;
      continue;
    }

    const im = rest.match(identRe);
    if (im) {
      tokens.push({ t: "ident", v: im[0] });
      i += im[0].length;
      continue;
    }

    return undefined;
  }

  let cur: unknown;
  let pos = 0;

  if (tokens[pos]?.t !== "ident") return undefined;
  const first = (tokens[pos] as { t: "ident"; v: string }).v;
  pos++;

  if (first === "$") cur = globals;
  else cur = globals ? (globals as Record<string, unknown>)[first] : undefined;

  while (pos < tokens.length) {
    const tok = tokens[pos];

    if (tok.t === "dot") {
      pos++;
      const next = tokens[pos];
      if (!next) return cur;
      if (next.t !== "ident") return undefined;
      cur = getProp(cur, next.v);
      pos++;
      continue;
    }

    if (tok.t === "key") {
      cur = getProp(cur, tok.v);
      pos++;
      continue;
    }

    return undefined;
  }

  return cur;
}

function listKeys(value: unknown): string[] {
  if (value == null) return [];
  if (Array.isArray(value)) {
    const keys: string[] = [];
    for (let i = 0; i < Math.min(10, value.length); i++) keys.push(String(i));
    return keys;
  }
  if (typeof value === "object") {
    return Object.keys(value as Record<string, unknown>).filter((key) => !key.startsWith("__"));
  }
  return [];
}

function getProp(obj: unknown, key: string): unknown {
  if (obj == null) return undefined;
  try {
    return (obj as any)[key];
  } catch {
    return undefined;
  }
}

function needsQuotingAsIdentifier(name: string): boolean {
  return !/^[$A-Za-z_][$A-Za-z0-9_]*$/.test(name);
}

function quoteKey(key: string, quote: "'" | '"'): string {
  const escaped = escapeString(key, quote);
  if (!Number.isNaN(Number(escaped))) {
    return escaped;
  }
  const q: "'" | '"' = quote === "'" ? "'" : '"';
  return q + escapeString(key, q) + q;
}

function escapeString(s: string, quote: "'" | '"' = '"'): string {
  const escaped = String(s).replace(/\\/g, "\\\\");
  return quote === "'" ? escaped.replace(/'/g, "\\'") : escaped.replace(/"/g, '\\"');
}

function unescapeString(s: string): string {
  return String(s).replace(/\\(["'\\])/g, "$1");
}

function isProbablyInsideString(left: string): boolean {
  const single = countUnescaped(left, "'");
  const dbl = countUnescaped(left, '"');
  return single % 2 === 1 || dbl % 2 === 1;
}

function countUnescaped(str: string, ch: "'" | '"'): number {
  let count = 0;
  for (let i = 0; i < str.length; i++) {
    if (str[i] === ch) {
      let backslashes = 0;
      for (let j = i - 1; j >= 0 && str[j] === "\\"; j--) backslashes++;
      if (backslashes % 2 === 0) count++;
    }
  }
  return count;
}
