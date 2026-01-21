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
  /** For node suggestions: the node ID (key used to access it) */
  nodeId?: string;
  /** For node suggestions: the human-readable node name */
  nodeName?: string;
  /** For function suggestions: short description of what the function does */
  description?: string;
  /** For function suggestions: example usage of the function */
  example?: string;
}

export interface GetSuggestionsOptions {
  includeFunctions?: boolean;
  includeGlobals?: boolean;
  limit?: number;
  allowInStrings?: boolean;
}

type ExprFunction = {
  name: string;
  snippet?: string;
  description: string;
  example: string;
};

/** Built-in functions (from expr-lang docs categories). */
export const EXPR_FUNCTIONS: readonly ExprFunction[] = [
  // String
  {
    name: "trim",
    snippet: "trim(${1:str}${2:, ${3:chars}})",
    description: "Removes whitespace from both ends of a string.",
    example: 'trim("  Hello  ") == "Hello"',
  },
  {
    name: "trimPrefix",
    snippet: "trimPrefix(${1:str}, ${2:prefix})",
    description: "Removes the specified prefix from a string.",
    example: 'trimPrefix("HelloWorld", "Hello") == "World"',
  },
  {
    name: "trimSuffix",
    snippet: "trimSuffix(${1:str}, ${2:suffix})",
    description: "Removes the specified suffix from a string.",
    example: 'trimSuffix("HelloWorld", "World") == "Hello"',
  },
  {
    name: "upper",
    snippet: "upper(${1:str})",
    description: "Converts all characters to uppercase.",
    example: 'upper("hello") == "HELLO"',
  },
  {
    name: "lower",
    snippet: "lower(${1:str})",
    description: "Converts all characters to lowercase.",
    example: 'lower("HELLO") == "hello"',
  },
  {
    name: "split",
    snippet: "split(${1:str}, ${2:delimiter}${3:, ${4:n}})",
    description: "Splits a string by delimiter into an array.",
    example: 'split("a,b,c", ",") == ["a", "b", "c"]',
  },
  {
    name: "splitAfter",
    snippet: "splitAfter(${1:str}, ${2:delimiter}${3:, ${4:n}})",
    description: "Splits a string after each delimiter.",
    example: 'splitAfter("a,b,c", ",") == ["a,", "b,", "c"]',
  },
  {
    name: "replace",
    snippet: "replace(${1:str}, ${2:old}, ${3:new})",
    description: "Replaces all occurrences of old with new.",
    example: 'replace("hello", "l", "L") == "heLLo"',
  },
  {
    name: "repeat",
    snippet: "repeat(${1:str}, ${2:n})",
    description: "Repeats a string n times.",
    example: 'repeat("Hi", 3) == "HiHiHi"',
  },
  {
    name: "indexOf",
    snippet: "indexOf(${1:str}, ${2:substring})",
    description: "Returns index of first occurrence, or -1.",
    example: 'indexOf("apple pie", "pie") == 6',
  },
  {
    name: "lastIndexOf",
    snippet: "lastIndexOf(${1:str}, ${2:substring})",
    description: "Returns index of last occurrence, or -1.",
    example: 'lastIndexOf("apple apple", "apple") == 6',
  },
  {
    name: "hasPrefix",
    snippet: "hasPrefix(${1:str}, ${2:prefix})",
    description: "Returns true if string starts with prefix.",
    example: 'hasPrefix("HelloWorld", "Hello") == true',
  },
  {
    name: "hasSuffix",
    snippet: "hasSuffix(${1:str}, ${2:suffix})",
    description: "Returns true if string ends with suffix.",
    example: 'hasSuffix("HelloWorld", "World") == true',
  },

  // Date
  {
    name: "now",
    snippet: "now()",
    description: "Returns the current date and time.",
    example: "now().Year() == 2024",
  },
  {
    name: "duration",
    snippet: "duration(${1:str})",
    description: "Parses a duration string (ns, us, ms, s, m, h).",
    example: 'duration("1h").Seconds() == 3600',
  },
  {
    name: "date",
    snippet: "date(${1:str}${2:, ${3:format}}${4:, ${5:timezone}})",
    description: "Parses a date string with optional format.",
    example: 'date("2023-08-14").Year() == 2023',
  },
  {
    name: "timezone",
    snippet: "timezone(${1:str})",
    description: "Returns a timezone by name.",
    example: 'timezone("Europe/Zurich")',
  },

  // Number
  {
    name: "max",
    snippet: "max(${1:n1}, ${2:n2})",
    description: "Returns the larger of two numbers.",
    example: "max(5, 7) == 7",
  },
  {
    name: "min",
    snippet: "min(${1:n1}, ${2:n2})",
    description: "Returns the smaller of two numbers.",
    example: "min(5, 7) == 5",
  },
  {
    name: "abs",
    snippet: "abs(${1:n})",
    description: "Returns the absolute value.",
    example: "abs(-5) == 5",
  },
  {
    name: "ceil",
    snippet: "ceil(${1:n})",
    description: "Rounds up to the nearest integer.",
    example: "ceil(1.5) == 2.0",
  },
  {
    name: "floor",
    snippet: "floor(${1:n})",
    description: "Rounds down to the nearest integer.",
    example: "floor(1.5) == 1.0",
  },
  {
    name: "round",
    snippet: "round(${1:n})",
    description: "Rounds to the nearest integer.",
    example: "round(1.5) == 2.0",
  },

  // Array
  {
    name: "all",
    snippet: "all(${1:array}, ${2:predicate})",
    description: "Returns true if all elements satisfy the predicate.",
    example: "all([1, 2, 3], # > 0) == true",
  },
  {
    name: "any",
    snippet: "any(${1:array}, ${2:predicate})",
    description: "Returns true if any element satisfies the predicate.",
    example: "any([1, 2, 3], # > 2) == true",
  },
  {
    name: "one",
    snippet: "one(${1:array}, ${2:predicate})",
    description: "Returns true if exactly one element satisfies.",
    example: "one([1, 2, 3], # == 2) == true",
  },
  {
    name: "none",
    snippet: "none(${1:array}, ${2:predicate})",
    description: "Returns true if no elements satisfy the predicate.",
    example: "none([1, 2, 3], # > 5) == true",
  },
  {
    name: "map",
    snippet: "map(${1:array}, ${2:predicate})",
    description: "Transforms each element using the predicate.",
    example: "map([1, 2, 3], # * 2) == [2, 4, 6]",
  },
  {
    name: "filter",
    snippet: "filter(${1:array}, ${2:predicate})",
    description: "Returns elements that satisfy the predicate.",
    example: "filter([1, 2, 3], # > 1) == [2, 3]",
  },
  {
    name: "find",
    snippet: "find(${1:array}, ${2:predicate})",
    description: "Returns first element that satisfies predicate.",
    example: "find([1, 2, 3], # > 1) == 2",
  },
  {
    name: "findIndex",
    snippet: "findIndex(${1:array}, ${2:predicate})",
    description: "Returns index of first matching element.",
    example: "findIndex([1, 2, 3], # > 1) == 1",
  },
  {
    name: "findLast",
    snippet: "findLast(${1:array}, ${2:predicate})",
    description: "Returns last element that satisfies predicate.",
    example: "findLast([1, 2, 3], # > 1) == 3",
  },
  {
    name: "findLastIndex",
    snippet: "findLastIndex(${1:array}, ${2:predicate})",
    description: "Returns index of last matching element.",
    example: "findLastIndex([1, 2, 3], # > 1) == 2",
  },
  {
    name: "groupBy",
    snippet: "groupBy(${1:array}, ${2:predicate})",
    description: "Groups elements by predicate result.",
    example: "groupBy(users, .Age)",
  },
  {
    name: "count",
    snippet: "count(${1:array}${2:, ${3:predicate}})",
    description: "Counts elements satisfying the predicate.",
    example: "count([1, 2, 3], # > 1) == 2",
  },
  {
    name: "concat",
    snippet: "concat(${1:array1}, ${2:array2}${3:, ${4:...}})",
    description: "Concatenates two or more arrays.",
    example: "concat([1, 2], [3, 4]) == [1, 2, 3, 4]",
  },
  {
    name: "flatten",
    snippet: "flatten(${1:array})",
    description: "Flattens nested arrays into one level.",
    example: "flatten([[1, 2], [3]]) == [1, 2, 3]",
  },
  {
    name: "uniq",
    snippet: "uniq(${1:array})",
    description: "Removes duplicate elements.",
    example: "uniq([1, 2, 2, 3]) == [1, 2, 3]",
  },
  {
    name: "join",
    snippet: "join(${1:array}${2:, ${3:delimiter}})",
    description: "Joins array elements into a string.",
    example: 'join(["a", "b"], ",") == "a,b"',
  },
  {
    name: "reduce",
    snippet: "reduce(${1:array}, ${2:predicate}${3:, ${4:initialValue}})",
    description: "Reduces array to single value using accumulator.",
    example: "reduce([1, 2, 3], #acc + #, 0) == 6",
  },
  {
    name: "sum",
    snippet: "sum(${1:array}${2:, ${3:predicate}})",
    description: "Returns sum of all numbers in array.",
    example: "sum([1, 2, 3]) == 6",
  },
  {
    name: "mean",
    snippet: "mean(${1:array})",
    description: "Returns average of all numbers.",
    example: "mean([1, 2, 3]) == 2.0",
  },
  {
    name: "median",
    snippet: "median(${1:array})",
    description: "Returns median of all numbers.",
    example: "median([1, 2, 3]) == 2.0",
  },
  {
    name: "first",
    snippet: "first(${1:array})",
    description: "Returns first element, or nil if empty.",
    example: "first([1, 2, 3]) == 1",
  },
  {
    name: "last",
    snippet: "last(${1:array})",
    description: "Returns last element, or nil if empty.",
    example: "last([1, 2, 3]) == 3",
  },
  {
    name: "take",
    snippet: "take(${1:array}, ${2:n})",
    description: "Returns first n elements.",
    example: "take([1, 2, 3, 4], 2) == [1, 2]",
  },
  {
    name: "reverse",
    snippet: "reverse(${1:array})",
    description: "Returns array in reverse order.",
    example: "reverse([1, 2, 3]) == [3, 2, 1]",
  },
  {
    name: "sort",
    snippet: "sort(${1:array}${2:, ${3:order}})",
    description: "Sorts array in ascending or descending order.",
    example: "sort([3, 1, 2]) == [1, 2, 3]",
  },
  {
    name: "sortBy",
    snippet: "sortBy(${1:array}${2:, ${3:predicate}}${4:, ${5:order}})",
    description: "Sorts array by predicate result.",
    example: 'sortBy(users, .Age, "desc")',
  },

  // Map
  {
    name: "keys",
    snippet: "keys(${1:map})",
    description: "Returns array of map keys.",
    example: 'keys({a: 1, b: 2}) == ["a", "b"]',
  },
  {
    name: "values",
    snippet: "values(${1:map})",
    description: "Returns array of map values.",
    example: "values({a: 1, b: 2}) == [1, 2]",
  },

  // Type conversion
  {
    name: "type",
    snippet: "type(${1:v})",
    description: "Returns the type name of a value.",
    example: 'type(42) == "int"',
  },
  {
    name: "int",
    snippet: "int(${1:v})",
    description: "Converts value to integer.",
    example: 'int("123") == 123',
  },
  {
    name: "float",
    snippet: "float(${1:v})",
    description: "Converts value to float.",
    example: 'float("1.5") == 1.5',
  },
  {
    name: "string",
    snippet: "string(${1:v})",
    description: "Converts value to string.",
    example: 'string(123) == "123"',
  },
  {
    name: "toJSON",
    snippet: "toJSON(${1:v})",
    description: "Converts value to JSON string.",
    example: "toJSON({a: 1}) == '{\"a\":1}'",
  },
  {
    name: "fromJSON",
    snippet: "fromJSON(${1:v})",
    description: "Parses JSON string to value.",
    example: "fromJSON('{\"a\":1}') == {a: 1}",
  },
  {
    name: "toBase64",
    snippet: "toBase64(${1:v})",
    description: "Encodes string to Base64.",
    example: 'toBase64("Hello") == "SGVsbG8="',
  },
  {
    name: "fromBase64",
    snippet: "fromBase64(${1:v})",
    description: "Decodes Base64 to string.",
    example: 'fromBase64("SGVsbG8=") == "Hello"',
  },
  {
    name: "toPairs",
    snippet: "toPairs(${1:map})",
    description: "Converts map to key-value pairs array.",
    example: 'toPairs({a: 1}) == [["a", 1]]',
  },
  {
    name: "fromPairs",
    snippet: "fromPairs(${1:array})",
    description: "Converts key-value pairs to map.",
    example: 'fromPairs([["a", 1]]) == {a: 1}',
  },

  // Misc
  {
    name: "len",
    snippet: "len(${1:v})",
    description: "Returns length of array, map, or string.",
    example: 'len("hello") == 5',
  },
  {
    name: "get",
    snippet: "get(${1:v}, ${2:index})",
    description: "Gets element by index/key, or nil if missing.",
    example: "get([1, 2, 3], 1) == 2",
  },

  // Bitwise
  {
    name: "bitand",
    snippet: "bitand(${1:a}, ${2:b})",
    description: "Bitwise AND operation.",
    example: "bitand(0b1010, 0b1100) == 0b1000",
  },
  {
    name: "bitor",
    snippet: "bitor(${1:a}, ${2:b})",
    description: "Bitwise OR operation.",
    example: "bitor(0b1010, 0b1100) == 0b1110",
  },
  {
    name: "bitxor",
    snippet: "bitxor(${1:a}, ${2:b})",
    description: "Bitwise XOR operation.",
    example: "bitxor(0b1010, 0b1100) == 0b0110",
  },
  {
    name: "bitnand",
    snippet: "bitnand(${1:a}, ${2:b})",
    description: "Bitwise AND NOT operation.",
    example: "bitnand(0b1010, 0b1100) == 0b0010",
  },
  {
    name: "bitnot",
    snippet: "bitnot(${1:a})",
    description: "Bitwise NOT operation.",
    example: "bitnot(0b1010) == -0b1011",
  },
  {
    name: "bitshl",
    snippet: "bitshl(${1:a}, ${2:b})",
    description: "Left shift operation.",
    example: "bitshl(0b101, 2) == 0b10100",
  },
  {
    name: "bitshr",
    snippet: "bitshr(${1:a}, ${2:b})",
    description: "Right shift operation.",
    example: "bitshr(0b101, 1) == 0b10",
  },
  {
    name: "bitushr",
    snippet: "bitushr(${1:a}, ${2:b})",
    description: "Unsigned right shift operation.",
    example: "bitushr(-5, 2) == 4611686018427387902",
  },
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
      const nodeName = getNodeName(globals, k, v);
      return {
        label: `["${k}"]`,
        kind: "variable",
        insertText: `$[${quoteKey(k, envTrigger.quote)}]${tailDot}`,
        detail: getValueTypeLabel(v),
        labelDetail: formatNodeNameLabel(nodeName),
        nodeId: nodeName ? k : undefined,
        nodeName: nodeName,
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
      .map((k) => {
        const v = (globals as Record<string, unknown>)[k];
        const nodeName = getNodeName(globals, k, v);
        return {
          label: `["${k}"]`,
          kind: "variable",
          insertText: quoteKey(k, bracketCtx.quote), // only the key token
          detail: getValueTypeLabel(v),
          labelDetail: formatNodeNameLabel(nodeName),
          nodeId: nodeName ? k : undefined,
          nodeName: nodeName,
        };
      });
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
          detail: "function",
          description: fn.description,
          example: fn.example,
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
