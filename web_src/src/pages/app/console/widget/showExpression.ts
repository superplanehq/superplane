/**
 * Tiny expression evaluator for Dashboard widget `show` / row filter clauses.
 *
 * Intentionally minimal: we support comparisons (`==`, `!=`, `>`, `<`, `>=`,
 * `<=`), logical operators (`&&`, `||`, `!`), parentheses, field paths, and
 * primitive literals (strings, numbers, booleans, `null`). Anything else
 * throws. We never call `eval` or `new Function` so widgets imported from
 * untrusted YAML can't escape the sandbox.
 *
 * Field paths are evaluated against a single `row` object passed at runtime.
 */

import { getValueAtPath } from "./fieldPath";

type Token =
  | { kind: "number"; value: number }
  | { kind: "string"; value: string }
  | { kind: "bool"; value: boolean }
  | { kind: "null" }
  | { kind: "ident"; value: string }
  | { kind: "op"; value: string }
  | { kind: "lparen" }
  | { kind: "rparen" };

const OPERATORS = ["==", "!=", ">=", "<=", "&&", "||", "!", ">", "<"];

interface TokenRead {
  token?: Token;
  nextIndex: number;
}

function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  while (i < input.length) {
    const read = readToken(input, i);
    if (read.token) tokens.push(read.token);
    i = read.nextIndex;
  }
  return tokens;
}

function readToken(input: string, index: number): TokenRead {
  const ch = input[index];
  if (isWhitespace(ch)) return { nextIndex: index + 1 };
  if (ch === "(") return { token: { kind: "lparen" }, nextIndex: index + 1 };
  if (ch === ")") return { token: { kind: "rparen" }, nextIndex: index + 1 };
  if (ch === '"' || ch === "'") return readStringToken(input, index, ch);
  if (isNumberStart(input, index)) return readNumberToken(input, index);

  const operator = readOperator(input, index);
  if (operator) return { token: { kind: "op", value: operator }, nextIndex: index + operator.length };
  if (isIdentifierStart(ch)) return readIdentifierToken(input, index);

  throw new Error(`Unexpected character '${ch}' in expression: ${input}`);
}

function readStringToken(input: string, index: number, quote: string): TokenRead {
  let j = index + 1;
  let buffer = "";
  while (j < input.length && input[j] !== quote) {
    if (input[j] === "\\" && j + 1 < input.length) {
      buffer += input[j + 1];
      j += 2;
      continue;
    }
    buffer += input[j];
    j++;
  }
  if (j >= input.length) throw new Error(`Unterminated string in expression: ${input}`);
  return { token: { kind: "string", value: buffer }, nextIndex: j + 1 };
}

function readNumberToken(input: string, index: number): TokenRead {
  let j = index + 1;
  while (j < input.length && /[0-9.]/.test(input[j])) j++;
  return { token: { kind: "number", value: Number(input.slice(index, j)) }, nextIndex: j };
}

function readOperator(input: string, index: number): string | undefined {
  return OPERATORS.find((op) => input.slice(index, index + op.length) === op);
}

function readIdentifierToken(input: string, index: number): TokenRead {
  let j = index + 1;
  while (j < input.length && /[A-Za-z0-9_.$[\]]/.test(input[j])) j++;
  const raw = input.slice(index, j);
  return { token: tokenForIdentifier(raw), nextIndex: j };
}

function tokenForIdentifier(raw: string): Token {
  if (raw === "true") return { kind: "bool", value: true };
  if (raw === "false") return { kind: "bool", value: false };
  if (raw === "null") return { kind: "null" };
  return { kind: "ident", value: raw };
}

function isWhitespace(ch: string | undefined): boolean {
  return ch === " " || ch === "\t" || ch === "\n";
}

function isNumberStart(input: string, index: number): boolean {
  const ch = input[index];
  return /[0-9]/.test(ch) || (ch === "-" && /[0-9]/.test(input[index + 1] ?? ""));
}

function isIdentifierStart(ch: string | undefined): boolean {
  return Boolean(ch && /[A-Za-z_$]/.test(ch));
}

interface ParserState {
  index: number;
  tokens: Token[];
}

function peek(state: ParserState): Token | undefined {
  return state.tokens[state.index];
}
function consume(state: ParserState): Token {
  return state.tokens[state.index++];
}

function parseOr(state: ParserState, row: unknown): unknown {
  let left = parseAnd(state, row);
  while (peek(state)?.kind === "op" && (peek(state) as { value: string }).value === "||") {
    consume(state);
    const right = parseAnd(state, row);
    left = Boolean(left) || Boolean(right);
  }
  return left;
}

function parseAnd(state: ParserState, row: unknown): unknown {
  let left = parseComparison(state, row);
  while (peek(state)?.kind === "op" && (peek(state) as { value: string }).value === "&&") {
    consume(state);
    const right = parseComparison(state, row);
    left = Boolean(left) && Boolean(right);
  }
  return left;
}

function parseComparison(state: ParserState, row: unknown): unknown {
  let left = parseUnary(state, row);
  const cmpOps = new Set(["==", "!=", ">", "<", ">=", "<="]);
  while (peek(state)?.kind === "op" && cmpOps.has((peek(state) as { value: string }).value)) {
    const op = (consume(state) as { value: string }).value;
    const right = parseUnary(state, row);
    left = compare(op, left, right);
  }
  return left;
}

function parseUnary(state: ParserState, row: unknown): unknown {
  const tok = peek(state);
  if (tok?.kind === "op" && tok.value === "!") {
    consume(state);
    return !parseUnary(state, row);
  }
  return parsePrimary(state, row);
}

function parsePrimary(state: ParserState, row: unknown): unknown {
  const tok = consume(state);
  if (!tok) throw new Error("Unexpected end of expression");
  if (tok.kind === "lparen") {
    const inner = parseOr(state, row);
    const closing = consume(state);
    if (!closing || closing.kind !== "rparen") throw new Error("Expected closing ')'");
    return inner;
  }
  if (tok.kind === "number" || tok.kind === "bool") return tok.value;
  if (tok.kind === "null") return null;
  if (tok.kind === "string") return tok.value;
  if (tok.kind === "ident") {
    // `row.foo.bar` and bare `foo.bar` both refer to the row.
    const path = tok.value.startsWith("row.") ? tok.value.slice(4) : tok.value;
    return getValueAtPath(row, path);
  }
  throw new Error(`Unexpected token: ${JSON.stringify(tok)}`);
}

function compare(op: string, left: unknown, right: unknown): boolean {
  switch (op) {
    case "==":
      return valuesEqual(left, right);
    case "!=":
      return !valuesEqual(left, right);
    case ">":
      return (left as number) > (right as number);
    case "<":
      return (left as number) < (right as number);
    case ">=":
      return (left as number) >= (right as number);
    case "<=":
      return (left as number) <= (right as number);
    default:
      throw new Error(`Unknown comparison: ${op}`);
  }
}

function valuesEqual(left: unknown, right: unknown): boolean {
  if (left === right) return true;
  return normalizeComparableScalar(left) === normalizeComparableScalar(right);
}

function normalizeComparableScalar(value: unknown): unknown {
  if (typeof value !== "string") return value;
  const trimmed = value.trim();
  if (trimmed === "") return value;
  if (trimmed === "true") return true;
  if (trimmed === "false") return false;
  if (trimmed === "null") return null;

  const numberValue = Number(trimmed);
  return Number.isFinite(numberValue) ? numberValue : value;
}

/** Discriminated result of parsing/evaluating a mini expression. */
export type ShowEvalResult = { ok: true; value: boolean } | { ok: false; error: string };

/**
 * Evaluate the mini expression against `row`, surfacing parse/eval failures
 * instead of swallowing them. Callers that can fall back to a richer evaluator
 * (e.g. CEL) use this to distinguish "the legacy tokenizer can't parse this"
 * from "the expression is legitimately false".
 */
export function tryEvaluateShow(expression: string, row: unknown): ShowEvalResult {
  try {
    const tokens = tokenize(expression);
    const state: ParserState = { index: 0, tokens };
    const result = parseOr(state, row);
    if (state.index < tokens.length) {
      throw new Error("Trailing tokens after expression");
    }
    return { ok: true, value: Boolean(result) };
  } catch (err) {
    return { ok: false, error: (err as Error).message };
  }
}

/**
 * Evaluate the given expression against a row context. Returns a boolean (the
 * truthiness of the resulting value). On parse error this logs to console and
 * returns `defaultValue` (defaults to `true`), so widgets remain functional
 * even when authoring mistakes are present.
 */
export function evaluateShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  const result = tryEvaluateShow(expression, row);
  if (result.ok) return result.value;
  if (typeof console !== "undefined") {
    console.warn(`Dashboard widget expression failed: ${result.error}`);
  }
  return defaultValue;
}
