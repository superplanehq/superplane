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
  /** For function/node suggestions: short description */
  description?: string;
  /** For function suggestions: example usage of the function */
  example?: string;
  /** For node suggestions: component/trigger type label */
  componentType?: string;
  /** For $ selector: number of available nodes */
  nodeCount?: number;
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
  {
    name: "root",
    snippet: "root().",
    description: "Returns the root payload that started the run.",
    example: "root().github.ref",
  },
  {
    name: "previous",
    snippet: "previous().",
    description:
      "Returns the payload from the immediate predecessor that emitted this event. Provide depth to walk upstream.",
    example: "previous(2).data.image.version",
  },
  // String
  {
    name: "trim",
    snippet: "trim(${1:str}${2:, ${3:chars}})",
    description:
      "Removes whitespace from both ends of a string. If the optional chars argument is given, removes those characters instead.",
    example: 'trim("  Hello  ") == "Hello"',
  },
  {
    name: "trimPrefix",
    snippet: "trimPrefix(${1:str}, ${2:prefix})",
    description: "Removes the specified prefix from the string if it starts with that prefix.",
    example: 'trimPrefix("HelloWorld", "Hello") == "World"',
  },
  {
    name: "trimSuffix",
    snippet: "trimSuffix(${1:str}, ${2:suffix})",
    description: "Removes the specified suffix from the string if it ends with that suffix.",
    example: 'trimSuffix("HelloWorld", "World") == "Hello"',
  },
  {
    name: "upper",
    snippet: "upper(${1:str})",
    description: "Converts all characters in the string to uppercase.",
    example: 'upper("hello") == "HELLO"',
  },
  {
    name: "lower",
    snippet: "lower(${1:str})",
    description: "Converts all characters in the string to lowercase.",
    example: 'lower("HELLO") == "hello"',
  },
  {
    name: "split",
    snippet: "split(${1:str}, ${2:delimiter}${3:, ${4:n}})",
    description:
      "Splits the string at each instance of the delimiter and returns an array of substrings. Optional n limits the number of splits.",
    example: 'split("a,b,c", ",") == ["a", "b", "c"]',
  },
  {
    name: "splitAfter",
    snippet: "splitAfter(${1:str}, ${2:delimiter}${3:, ${4:n}})",
    description: "Splits the string after each instance of the delimiter. Optional n limits the number of splits.",
    example: 'splitAfter("a,b,c", ",") == ["a,", "b,", "c"]',
  },
  {
    name: "replace",
    snippet: "replace(${1:str}, ${2:old}, ${3:new})",
    description: "Replaces all occurrences of old in string with new.",
    example: 'replace("Hello World", "World", "Universe") == "Hello Universe"',
  },
  {
    name: "repeat",
    snippet: "repeat(${1:str}, ${2:n})",
    description: "Repeats the string n times.",
    example: 'repeat("Hi", 3) == "HiHiHi"',
  },
  {
    name: "indexOf",
    snippet: "indexOf(${1:str}, ${2:substring})",
    description: "Returns the index of the first occurrence of the substring in the string, or -1 if not found.",
    example: 'indexOf("apple pie", "pie") == 6',
  },
  {
    name: "lastIndexOf",
    snippet: "lastIndexOf(${1:str}, ${2:substring})",
    description: "Returns the index of the last occurrence of the substring in the string, or -1 if not found.",
    example: 'lastIndexOf("apple pie apple", "apple") == 10',
  },
  {
    name: "hasPrefix",
    snippet: "hasPrefix(${1:str}, ${2:prefix})",
    description: "Returns true if string starts with the given prefix.",
    example: 'hasPrefix("HelloWorld", "Hello") == true',
  },
  {
    name: "hasSuffix",
    snippet: "hasSuffix(${1:str}, ${2:suffix})",
    description: "Returns true if string ends with the given suffix.",
    example: 'hasSuffix("HelloWorld", "World") == true',
  },

  // Date
  {
    name: "now",
    snippet: "now()",
    description: "Returns the current date as a time.Time value.",
    example: "now().Year() == 2024",
  },
  {
    name: "duration",
    snippet: "duration(${1:str})",
    description: 'Returns a time.Duration value of the given string. Valid units: "ns", "us", "ms", "s", "m", "h".',
    example: 'duration("1h").Seconds() == 3600',
  },
  {
    name: "date",
    snippet: "date(${1:str}${2:, ${3:format}}${4:, ${5:timezone}})",
    description:
      "Converts the given string into a date. Optional format specifies the date format using Go time layout. Optional timezone specifies the timezone.",
    example: 'date("2023-08-14").Year() == 2023',
  },
  {
    name: "timezone",
    snippet: "timezone(${1:str})",
    description: "Returns a timezone by name. Use with date.In() to convert dates to different timezones.",
    example: 'timezone("Europe/Zurich")',
  },

  // Number
  {
    name: "max",
    snippet: "max(${1:n1}, ${2:n2})",
    description: "Returns the maximum of the two numbers.",
    example: "max(5, 7) == 7",
  },
  {
    name: "min",
    snippet: "min(${1:n1}, ${2:n2})",
    description: "Returns the minimum of the two numbers.",
    example: "min(5, 7) == 5",
  },
  {
    name: "abs",
    snippet: "abs(${1:n})",
    description: "Returns the absolute value of a number.",
    example: "abs(-5) == 5",
  },
  {
    name: "ceil",
    snippet: "ceil(${1:n})",
    description: "Returns the least integer value greater than or equal to x.",
    example: "ceil(1.5) == 2.0",
  },
  {
    name: "floor",
    snippet: "floor(${1:n})",
    description: "Returns the greatest integer value less than or equal to x.",
    example: "floor(1.5) == 1.0",
  },
  {
    name: "round",
    snippet: "round(${1:n})",
    description: "Returns the nearest integer, rounding half away from zero.",
    example: "round(1.5) == 2.0",
  },

  // Array
  {
    name: "all",
    snippet: "all(${1:array}, ${2:predicate})",
    description: "Returns true if all elements satisfy the predicate. If the array is empty, returns true.",
    example: "all([1, 2, 3], # > 0) == true",
  },
  {
    name: "any",
    snippet: "any(${1:array}, ${2:predicate})",
    description: "Returns true if any element satisfies the predicate. If the array is empty, returns false.",
    example: "any([1, 2, 3], # > 2) == true",
  },
  {
    name: "one",
    snippet: "one(${1:array}, ${2:predicate})",
    description: "Returns true if exactly one element satisfies the predicate. If the array is empty, returns false.",
    example: "one([1, 2, 3], # == 2) == true",
  },
  {
    name: "none",
    snippet: "none(${1:array}, ${2:predicate})",
    description:
      "Returns true if all elements do not satisfy the predicate (none satisfy). If the array is empty, returns true.",
    example: "none([1, 2, 3], # > 5) == true",
  },
  {
    name: "map",
    snippet: "map(${1:array}, ${2:predicate})",
    description: "Returns a new array by applying the predicate to each element of the array.",
    example: "map([1, 2, 3], # * 2) == [2, 4, 6]",
  },
  {
    name: "filter",
    snippet: "filter(${1:array}, ${2:predicate})",
    description: "Returns a new array by filtering elements of the array by the predicate.",
    example: "filter([1, 2, 3], # > 1) == [2, 3]",
  },
  {
    name: "find",
    snippet: "find(${1:array}, ${2:predicate})",
    description: "Finds the first element in an array that satisfies the predicate.",
    example: "find([1, 2, 3], # > 1) == 2",
  },
  {
    name: "findIndex",
    snippet: "findIndex(${1:array}, ${2:predicate})",
    description: "Finds the index of the first element in an array that satisfies the predicate.",
    example: "findIndex([1, 2, 3], # > 1) == 1",
  },
  {
    name: "findLast",
    snippet: "findLast(${1:array}, ${2:predicate})",
    description: "Finds the last element in an array that satisfies the predicate.",
    example: "findLast([1, 2, 3], # > 1) == 3",
  },
  {
    name: "findLastIndex",
    snippet: "findLastIndex(${1:array}, ${2:predicate})",
    description: "Finds the index of the last element in an array that satisfies the predicate.",
    example: "findLastIndex([1, 2, 3], # > 1) == 2",
  },
  {
    name: "groupBy",
    snippet: "groupBy(${1:array}, ${2:predicate})",
    description: "Groups the elements of an array by the result of the predicate.",
    example: "groupBy(users, .Age)",
  },
  {
    name: "count",
    snippet: "count(${1:array}${2:, ${3:predicate}})",
    description:
      "Returns the number of elements that satisfy the predicate. If no predicate, counts true elements in the array.",
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
    description: "Flattens a given array into a one-dimensional array.",
    example: "flatten([[1, 2], [3]]) == [1, 2, 3]",
  },
  {
    name: "uniq",
    snippet: "uniq(${1:array})",
    description: "Removes duplicates from an array.",
    example: "uniq([1, 2, 2, 3]) == [1, 2, 3]",
  },
  {
    name: "join",
    snippet: "join(${1:array}${2:, ${3:delimiter}})",
    description:
      "Joins an array of strings into a single string with the given delimiter. If no delimiter, uses empty string.",
    example: 'join(["a", "b"], ",") == "a,b"',
  },
  {
    name: "reduce",
    snippet: "reduce(${1:array}, ${2:predicate}${3:, ${4:initialValue}})",
    description:
      "Applies a predicate to each element, reducing the array to a single value. Uses #acc for accumulator and # for current element.",
    example: "reduce([1, 2, 3], #acc + #, 0) == 6",
  },
  {
    name: "sum",
    snippet: "sum(${1:array}${2:, ${3:predicate}})",
    description:
      "Returns the sum of all numbers in the array. If predicate is given, applies it to each element before summing.",
    example: "sum([1, 2, 3]) == 6",
  },
  {
    name: "mean",
    snippet: "mean(${1:array})",
    description: "Returns the average of all numbers in the array.",
    example: "mean([1, 2, 3]) == 2.0",
  },
  {
    name: "median",
    snippet: "median(${1:array})",
    description: "Returns the median of all numbers in the array.",
    example: "median([1, 2, 3]) == 2.0",
  },
  {
    name: "first",
    snippet: "first(${1:array})",
    description: "Returns the first element from an array. If the array is empty, returns nil.",
    example: "first([1, 2, 3]) == 1",
  },
  {
    name: "last",
    snippet: "last(${1:array})",
    description: "Returns the last element from an array. If the array is empty, returns nil.",
    example: "last([1, 2, 3]) == 3",
  },
  {
    name: "take",
    snippet: "take(${1:array}, ${2:n})",
    description:
      "Returns the first n elements from an array. If array has fewer than n elements, returns the whole array.",
    example: "take([1, 2, 3, 4], 2) == [1, 2]",
  },
  {
    name: "reverse",
    snippet: "reverse(${1:array})",
    description: "Returns a new reversed copy of the array.",
    example: "reverse([1, 2, 3]) == [3, 2, 1]",
  },
  {
    name: "sort",
    snippet: "sort(${1:array}${2:, ${3:order}})",
    description: 'Sorts an array in ascending order. Optional order argument: "asc" or "desc".',
    example: "sort([3, 1, 2]) == [1, 2, 3]",
  },
  {
    name: "sortBy",
    snippet: "sortBy(${1:array}${2:, ${3:predicate}}${4:, ${5:order}})",
    description: 'Sorts an array by the result of the predicate. Optional order argument: "asc" or "desc".',
    example: 'sortBy(users, .Age, "desc")',
  },

  // Map
  {
    name: "keys",
    snippet: "keys(${1:map})",
    description: "Returns an array containing the keys of the map.",
    example: 'keys({a: 1, b: 2}) == ["a", "b"]',
  },
  {
    name: "values",
    snippet: "values(${1:map})",
    description: "Returns an array containing the values of the map.",
    example: "values({a: 1, b: 2}) == [1, 2]",
  },

  // Type conversion
  {
    name: "type",
    snippet: "type(${1:v})",
    description:
      "Returns the type of the given value: nil, bool, int, uint, float, string, array, map, or the struct name.",
    example: 'type(42) == "int"',
  },
  {
    name: "int",
    snippet: "int(${1:v})",
    description: "Returns the integer value of a number or a string.",
    example: 'int("123") == 123',
  },
  {
    name: "float",
    snippet: "float(${1:v})",
    description: "Returns the float value of a number or a string.",
    example: 'float("1.5") == 1.5',
  },
  {
    name: "string",
    snippet: "string(${1:v})",
    description: "Converts the given value into a string representation.",
    example: 'string(123) == "123"',
  },
  {
    name: "toJSON",
    snippet: "toJSON(${1:v})",
    description: "Converts the given value to its JSON string representation.",
    example: "toJSON({a: 1}) == '{\"a\":1}'",
  },
  {
    name: "fromJSON",
    snippet: "fromJSON(${1:v})",
    description: "Parses the given JSON string and returns the corresponding value.",
    example: "fromJSON('{\"a\":1}') == {a: 1}",
  },
  {
    name: "toBase64",
    snippet: "toBase64(${1:v})",
    description: "Encodes the string into Base64 format.",
    example: 'toBase64("Hello World") == "SGVsbG8gV29ybGQ="',
  },
  {
    name: "fromBase64",
    snippet: "fromBase64(${1:v})",
    description: "Decodes the Base64 encoded string back to its original form.",
    example: 'fromBase64("SGVsbG8gV29ybGQ=") == "Hello World"',
  },
  {
    name: "toPairs",
    snippet: "toPairs(${1:map})",
    description: "Converts a map to an array of key-value pairs.",
    example: 'toPairs({a: 1}) == [["a", 1]]',
  },
  {
    name: "fromPairs",
    snippet: "fromPairs(${1:array})",
    description: "Converts an array of key-value pairs to a map.",
    example: 'fromPairs([["a", 1]]) == {a: 1}',
  },

  // Misc
  {
    name: "len",
    snippet: "len(${1:v})",
    description: "Returns the length of an array, a map, or a string.",
    example: 'len("hello") == 5',
  },
  {
    name: "get",
    snippet: "get(${1:v}, ${2:index})",
    description:
      "Retrieves the element at the specified index from an array or map. Returns nil if index is out of range or key does not exist.",
    example: "get([1, 2, 3], 1) == 2",
  },

  // Bitwise
  {
    name: "bitand",
    snippet: "bitand(${1:a}, ${2:b})",
    description: "Returns the values resulting from the bitwise AND operation.",
    example: "bitand(0b1010, 0b1100) == 0b1000",
  },
  {
    name: "bitor",
    snippet: "bitor(${1:a}, ${2:b})",
    description: "Returns the values resulting from the bitwise OR operation.",
    example: "bitor(0b1010, 0b1100) == 0b1110",
  },
  {
    name: "bitxor",
    snippet: "bitxor(${1:a}, ${2:b})",
    description: "Returns the values resulting from the bitwise XOR operation.",
    example: "bitxor(0b1010, 0b1100) == 0b0110",
  },
  {
    name: "bitnand",
    snippet: "bitnand(${1:a}, ${2:b})",
    description: "Returns the values resulting from the bitwise AND NOT operation.",
    example: "bitnand(0b1010, 0b1100) == 0b0010",
  },
  {
    name: "bitnot",
    snippet: "bitnot(${1:a})",
    description: "Returns the values resulting from the bitwise NOT operation.",
    example: "bitnot(0b1010) == -0b1011",
  },
  {
    name: "bitshl",
    snippet: "bitshl(${1:a}, ${2:b})",
    description: "Returns the values resulting from the Left Shift operation.",
    example: "bitshl(0b101101, 2) == 0b10110100",
  },
  {
    name: "bitshr",
    snippet: "bitshr(${1:a}, ${2:b})",
    description: "Returns the values resulting from the Right Shift operation.",
    example: "bitshr(0b101101, 2) == 0b1011",
  },
  {
    name: "bitushr",
    snippet: "bitushr(${1:a}, ${2:b})",
    description: "Returns the values resulting from the unsigned Right Shift operation.",
    example: "bitushr(-0b101, 2) == 4611686018427387902",
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
      const metadata = getNodeMetadata(globals, k, v);
      const labelDetail = metadata.nodeName ? formatNodeNameLabel(metadata.nodeName) : undefined;
      return {
        label: `["${k}"]`,
        kind: "variable",
        insertText: `$[${quoteKey(k, envTrigger.quote)}]${tailDot}`,
        detail: getValueTypeLabel(v),
        labelDetail,
        nodeName: metadata.nodeName,
        componentType: metadata.componentType,
        description: metadata.description,
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
        const metadata = getNodeMetadata(globals, k, v);
        const labelDetail = metadata.nodeName ? formatNodeNameLabel(metadata.nodeName) : undefined;
        return {
          label: `["${k}"]`,
          kind: "variable",
          insertText: quoteKey(k, bracketCtx.quote), // only the key token
          detail: getValueTypeLabel(v),
          labelDetail,
          nodeName: metadata.nodeName,
          componentType: metadata.componentType,
          description: metadata.description,
        };
      });
  }

  // 2) Now it's safe to suppress suggestions inside normal strings
  if (!allowInStrings && isProbablyInsideString(left)) return [];

  // 3) Dot completion
  const dotCtx = detectDotContext(left);
  if (dotCtx) {
    const { baseExpr, memberPrefix, operator, isFunctionCall } = dotCtx;
    const mp = (memberPrefix ?? "").toLowerCase();

    // Handle function call method suggestions (e.g., now().Year())
    if (isFunctionCall) {
      const methods = getFunctionReturnMethods(baseExpr);
      if (methods.length > 0) {
        return methods
          .filter((m) => m.name.toLowerCase().startsWith(mp) && mp !== m.name.toLowerCase())
          .slice(0, limit)
          .map((m) => ({
            label: m.name,
            kind: "function" as const,
            insertText: m.snippet ?? `${m.name}()`,
            detail: m.returnType,
            description: m.description,
          }));
      }
    }

    const resolvableBase = isFunctionCall ? baseExpr : extractTailPathExpression(baseExpr);
    const target = resolveExprToValue(resolvableBase, globals);

    const keys = listKeys(target);

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
      const nodeCount = listGlobalKeys(globals).length;
      out.push({
        label: "$",
        kind: "variable",
        insertText: "$",
        detail: getValueTypeLabel(globals),
        description: "Root selector for accessing payload data from all connected components.",
        nodeCount,
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

type MethodInfo = {
  name: string;
  snippet?: string;
  returnType: string;
  description: string;
};

// Date/Time methods available on now(), date(), duration() return values
const DATE_METHODS: MethodInfo[] = [
  { name: "Year", snippet: "Year()", returnType: "int", description: "Returns the year (e.g., 2024)" },
  { name: "Month", snippet: "Month()", returnType: "int", description: "Returns the month (1-12)" },
  { name: "Day", snippet: "Day()", returnType: "int", description: "Returns the day of the month (1-31)" },
  { name: "Hour", snippet: "Hour()", returnType: "int", description: "Returns the hour (0-23)" },
  { name: "Minute", snippet: "Minute()", returnType: "int", description: "Returns the minute (0-59)" },
  { name: "Second", snippet: "Second()", returnType: "int", description: "Returns the second (0-59)" },
  { name: "Weekday", snippet: "Weekday()", returnType: "int", description: "Returns the day of the week (0=Sunday)" },
  { name: "YearDay", snippet: "YearDay()", returnType: "int", description: "Returns the day of the year (1-366)" },
  { name: "Unix", snippet: "Unix()", returnType: "int", description: "Returns Unix timestamp in seconds" },
  {
    name: "UnixMilli",
    snippet: "UnixMilli()",
    returnType: "int",
    description: "Returns Unix timestamp in milliseconds",
  },
  {
    name: "UnixMicro",
    snippet: "UnixMicro()",
    returnType: "int",
    description: "Returns Unix timestamp in microseconds",
  },
  { name: "UnixNano", snippet: "UnixNano()", returnType: "int", description: "Returns Unix timestamp in nanoseconds" },
  {
    name: "Format",
    snippet: 'Format("${1:2006-01-02}")',
    returnType: "string",
    description: "Formats the date using Go time layout",
  },
  {
    name: "Add",
    snippet: 'Add(duration("${1:1h}"))',
    returnType: "time",
    description: "Adds a duration to the time",
  },
  {
    name: "Sub",
    snippet: "Sub(${1:time})",
    returnType: "duration",
    description: "Returns duration between two times",
  },
  { name: "Before", snippet: "Before(${1:time})", returnType: "bool", description: "Reports whether time is before t" },
  { name: "After", snippet: "After(${1:time})", returnType: "bool", description: "Reports whether time is after t" },
  { name: "Equal", snippet: "Equal(${1:time})", returnType: "bool", description: "Reports whether times are equal" },
  {
    name: "In",
    snippet: 'In(timezone("${1:UTC}"))',
    returnType: "time",
    description: "Returns time in the specified timezone",
  },
  { name: "UTC", snippet: "UTC()", returnType: "time", description: "Returns time in UTC timezone" },
  { name: "Local", snippet: "Local()", returnType: "time", description: "Returns time in local timezone" },
  { name: "IsZero", snippet: "IsZero()", returnType: "bool", description: "Reports whether time is zero value" },
  {
    name: "Round",
    snippet: 'Round(duration("${1:1h}"))',
    returnType: "time",
    description: "Rounds time to nearest duration",
  },
  {
    name: "Truncate",
    snippet: 'Truncate(duration("${1:1h}"))',
    returnType: "time",
    description: "Truncates time to duration",
  },
];

// Duration methods available on duration() return values
const DURATION_METHODS: MethodInfo[] = [
  { name: "Hours", snippet: "Hours()", returnType: "float", description: "Returns duration as hours" },
  { name: "Minutes", snippet: "Minutes()", returnType: "float", description: "Returns duration as minutes" },
  { name: "Seconds", snippet: "Seconds()", returnType: "float", description: "Returns duration as seconds" },
  {
    name: "Milliseconds",
    snippet: "Milliseconds()",
    returnType: "int",
    description: "Returns duration as milliseconds",
  },
  {
    name: "Microseconds",
    snippet: "Microseconds()",
    returnType: "int",
    description: "Returns duration as microseconds",
  },
  { name: "Nanoseconds", snippet: "Nanoseconds()", returnType: "int", description: "Returns duration as nanoseconds" },
  { name: "Abs", snippet: "Abs()", returnType: "duration", description: "Returns absolute value of duration" },
  {
    name: "Round",
    snippet: 'Round(duration("${1:1s}"))',
    returnType: "duration",
    description: "Rounds duration to nearest multiple",
  },
  {
    name: "Truncate",
    snippet: 'Truncate(duration("${1:1s}"))',
    returnType: "duration",
    description: "Truncates duration to multiple",
  },
];

// Map of function names to their return type methods
const FUNCTION_RETURN_METHODS: Record<string, MethodInfo[]> = {
  now: DATE_METHODS,
  date: DATE_METHODS,
  duration: DURATION_METHODS,
};

function getFunctionReturnMethods(funcName: string): MethodInfo[] {
  return FUNCTION_RETURN_METHODS[funcName.toLowerCase()] ?? [];
}

type EnvKeyTrigger = { quote: "'" | '"' };

function detectEnvKeyTrigger(left: string): EnvKeyTrigger | null {
  if (/\$\s*$/.test(left)) return { quote: '"' };
  if (/\$\s*\[\s*$/.test(left)) return { quote: '"' };
  return null;
}

function listGlobalKeys(globals: Record<string, unknown>): string[] {
  const keys = Object.keys(globals ?? {}).filter((key) => !key.startsWith("__"));
  const nodeNames = isRecord(globals) ? (globals as Record<string, unknown>).__nodeNames : undefined;

  if (!isRecord(nodeNames)) {
    return keys;
  }

  const nodeIds = new Set(Object.keys(nodeNames));
  const nameKeys = new Set<string>();

  for (const entry of Object.values(nodeNames)) {
    const name = extractNodeName(entry);
    if (name && !name.startsWith("__")) {
      nameKeys.add(name);
    }
  }

  const nonNodeKeys = keys.filter((key) => !nodeIds.has(key));
  return Array.from(new Set([...nonNodeKeys, ...nameKeys]));
}

function isExpandableValue(v: unknown): boolean {
  if (v === null || typeof v !== "object") return false;
  return Object.values(v as Record<string, unknown>).length > 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

type NodeMetadata = {
  nodeName?: string;
  componentType?: string;
  description?: string;
};

function getNodeMetadata(globals: Record<string, unknown>, key: string, value: unknown): NodeMetadata {
  const metadata: NodeMetadata = {};

  // Try to get nodeName from the value itself
  if (isRecord(value)) {
    const nodeName = value.__nodeName;
    if (typeof nodeName === "string" && nodeName.trim()) {
      metadata.nodeName = nodeName;
    }
  }

  // Get full metadata from __nodeNames
  const nodeNames = isRecord(globals) ? (globals as Record<string, unknown>).__nodeNames : undefined;
  if (isRecord(nodeNames)) {
    const entry = nodeNames[key];
    applyNodeEntryMetadata(metadata, entry);

    if (!metadata.nodeName) {
      for (const entryValue of Object.values(nodeNames)) {
        const entryName = extractNodeName(entryValue);
        if (!entryName || entryName !== key) {
          continue;
        }

        applyNodeEntryMetadata(metadata, entryValue);
        if (metadata.nodeName) {
          break;
        }
      }
    }
  }

  return metadata;
}

function formatNodeNameLabel(nodeName?: string): string | undefined {
  if (!nodeName) return undefined;
  return `- (${nodeName})`;
}

function extractNodeName(entry: unknown): string | undefined {
  if (typeof entry === "string") {
    return entry.trim() ? entry : undefined;
  }
  if (isRecord(entry)) {
    const name = entry.name ?? entry.nodeName ?? entry.label;
    if (typeof name === "string" && name.trim()) {
      return name;
    }
  }
  return undefined;
}

function applyNodeEntryMetadata(metadata: NodeMetadata, entry: unknown): void {
  if (typeof entry === "string" && entry.trim()) {
    metadata.nodeName = metadata.nodeName || entry;
    return;
  }
  if (!isRecord(entry)) {
    return;
  }

  const name = extractNodeName(entry);
  if (name) {
    metadata.nodeName = metadata.nodeName || name;
  }
  if (typeof entry.componentType === "string" && entry.componentType.trim()) {
    metadata.componentType = entry.componentType;
  }
  if (typeof entry.description === "string" && entry.description.trim()) {
    metadata.description = entry.description;
  }
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

type DotContext = { baseExpr: string; memberPrefix: string; operator: "." | "?."; isFunctionCall?: boolean };

function detectDotContext(left: string): DotContext | null {
  if (left.endsWith(" ")) {
    return null;
  }

  const tail = extractTailExpressionWithParens(left);
  const expr = tail.trim();
  if (!expr) return null;

  // Match expressions starting with $ (e.g., $["key"].prop)
  const dollarMatch = expr.match(/(\$.+?)(\?\.|\.)\s*([$A-Za-z_][$A-Za-z0-9_]*)?$/);
  if (dollarMatch) {
    let baseExpr = (dollarMatch[1] ?? "").trim();
    const operator = (dollarMatch[2] === "?." ? "?." : ".") as "." | "?.";
    const memberPrefix = (dollarMatch[3] ?? "").trim();
    if (!baseExpr) return null;

    if (baseExpr.includes("$")) {
      baseExpr = "$" + baseExpr.split("$").at(-1);
    }

    if (/^\d+(\.\d+)?$/.test(baseExpr)) return null;
    return { baseExpr, memberPrefix, operator };
  }

  // Match function calls (e.g., now(), date("2024-01-01"), duration("1h"))
  const funcMatch = expr.match(/((?:[a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\))(\?\.|\.)\s*([$A-Za-z_][$A-Za-z0-9_]*)?$/);
  if (funcMatch) {
    let funcName = extractTailExpressionWithParens(funcMatch[1]);
    funcName = extractSpecialFunctionCall(funcName);
    const operator = (funcMatch[2] === "?." ? "?." : ".") as "." | "?.";
    const memberPrefix = (funcMatch[3] ?? "").trim();
    if (!funcName) return null;
    return { baseExpr: funcName, memberPrefix, operator, isFunctionCall: true };
  }

  const tailCtx = findTailDotContext(expr);
  if (tailCtx) {
    return tailCtx;
  }

  return null;
}

type BracketKeyContext = { quote: "'" | '"'; partialKey: string };

function detectBracketKeyContext(left: string): BracketKeyContext | null {
  const m = left.match(/(?:\$\s*|\]\s*)\[\s*(['"])([^'"]*)$/);
  if (!m) return null;

  const quote = (m[1] === "'" ? "'" : '"') as "'" | '"';
  const partialKey = m[2] ?? "";
  return { quote, partialKey };
}

function extractTailPathExpression(expr: string | undefined | null): string {
  if (!expr) return "";
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

function findTailDotContext(input: string): DotContext | null {
  const s = input.trim();
  let i = s.length - 1;
  let bracketDepth = 0;
  let parenDepth = 0;
  let inSingle = false;
  let inDouble = false;

  const isEscaped = (idx: number): boolean => {
    let bs = 0;
    for (let j = idx - 1; j >= 0 && s[j] === "\\"; j--) bs++;
    return bs % 2 === 1;
  };

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
    if (ch === ")") {
      parenDepth++;
      continue;
    }
    if (ch === "(") {
      if (parenDepth == 0) {
        return null;
      }
      parenDepth = Math.max(0, parenDepth - 1);
      continue;
    }

    if (bracketDepth > 0 || parenDepth > 0) continue;

    if (ch === "." && i > 0 && /\d/.test(s[i - 1]) && i + 1 < s.length && /\d/.test(s[i + 1])) {
      continue;
    }

    if (ch === "." || (ch === "?" && s[i + 1] === ".")) {
      const operator = ch === "?" ? "?." : ".";
      const opStart = ch === "?" ? i : i;
      let baseExpr = s.slice(0, opStart).trim();
      baseExpr = extractTailExpressionWithParens(baseExpr);
      baseExpr = extractSpecialFunctionCall(baseExpr);
      const memberPrefix = s.slice(opStart + operator.length).trim();
      if (!baseExpr) return null;
      if (/^\d+(\.\d+)?$/.test(baseExpr)) return null;
      const isFunctionCall = baseExpr.includes("(");
      return { baseExpr, memberPrefix, operator: operator as "." | "?.", isFunctionCall };
    }
  }

  return null;
}

type Token = { t: "dot" } | { t: "ident"; v: string } | { t: "key"; v: string };

function extractTailExpressionWithParens(expr: string): string {
  const s = expr.trim();
  let i = s.length - 1;
  let bracketDepth = 0;
  let parenDepth = 0;
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
    if (ch === ")") {
      parenDepth++;
      continue;
    }
    if (ch === "(") {
      if (parenDepth === 0) {
        return s.slice(i + 1).trim();
      }
      parenDepth = Math.max(0, parenDepth - 1);
      continue;
    }

    if (bracketDepth > 0 || parenDepth > 0) continue;

    if (isStopChar(ch)) {
      return s.slice(i + 1).trim();
    }
  }

  return s;
}

function extractSpecialFunctionCall(expr: string): string {
  const matches = [...expr.matchAll(/(root\(\)|previous\([^)]*\))/g)];
  if (matches.length === 0) {
    return expr;
  }

  const lastMatch = matches[matches.length - 1];
  const matchValue = lastMatch?.[0];
  if (!matchValue || lastMatch?.index === undefined) {
    return expr;
  }

  const afterMatch = expr.slice(lastMatch.index + matchValue.length).trimStart();
  if (afterMatch.startsWith(".") || afterMatch.startsWith("?.")) {
    return expr;
  }

  return matchValue;
}

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
  expr = extractTailExpressionWithParens(expr);
  const normalized = normalizeSpecialFunctionExpr(expr);
  if (normalized === null) {
    return undefined;
  }
  expr = normalized;

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

function normalizeSpecialFunctionExpr(expr: string): string | null {
  const rootMatch = expr.match(/^root\(\)/);
  if (rootMatch) {
    return `__root${expr.slice(rootMatch[0].length)}`;
  }

  const previousMatch = expr.match(/^previous\(([^)]*)\)/);
  if (previousMatch) {
    const raw = (previousMatch[1] ?? "").trim();
    const depth = raw === "" ? 1 : Number(raw);
    if (!Number.isInteger(depth) || depth < 1) {
      return null;
    }

    return `__previousByDepth["${depth}"]${expr.slice(previousMatch[0].length)}`;
  }

  return expr;
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
