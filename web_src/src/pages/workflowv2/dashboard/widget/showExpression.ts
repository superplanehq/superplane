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

function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  while (i < input.length) {
    const ch = input[i];
    if (ch === " " || ch === "\t" || ch === "\n") {
      i++;
      continue;
    }
    if (ch === "(") {
      tokens.push({ kind: "lparen" });
      i++;
      continue;
    }
    if (ch === ")") {
      tokens.push({ kind: "rparen" });
      i++;
      continue;
    }
    if (ch === '"' || ch === "'") {
      const quote = ch;
      let j = i + 1;
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
      tokens.push({ kind: "string", value: buffer });
      i = j + 1;
      continue;
    }
    if (/[0-9]/.test(ch) || (ch === "-" && /[0-9]/.test(input[i + 1] ?? ""))) {
      let j = i + 1;
      while (j < input.length && /[0-9.]/.test(input[j])) j++;
      const value = Number(input.slice(i, j));
      tokens.push({ kind: "number", value });
      i = j;
      continue;
    }
    // Operators
    let matchedOp: string | undefined;
    for (const op of OPERATORS) {
      if (input.slice(i, i + op.length) === op) {
        matchedOp = op;
        break;
      }
    }
    if (matchedOp) {
      tokens.push({ kind: "op", value: matchedOp });
      i += matchedOp.length;
      continue;
    }
    if (/[A-Za-z_$]/.test(ch)) {
      let j = i + 1;
      while (j < input.length && /[A-Za-z0-9_.$[\]]/.test(input[j])) j++;
      const raw = input.slice(i, j);
      if (raw === "true") tokens.push({ kind: "bool", value: true });
      else if (raw === "false") tokens.push({ kind: "bool", value: false });
      else if (raw === "null") tokens.push({ kind: "null" });
      else tokens.push({ kind: "ident", value: raw });
      i = j;
      continue;
    }
    throw new Error(`Unexpected character '${ch}' in expression: ${input}`);
  }
  return tokens;
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
      // eslint-disable-next-line eqeqeq -- intentional loose equality so YAML strings/numbers compare naturally
      return left == right;
    case "!=":
      // eslint-disable-next-line eqeqeq
      return left != right;
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

/**
 * Evaluate the given expression against a row context. Returns a boolean (the
 * truthiness of the resulting value). On parse error this logs to console and
 * returns `defaultValue` (defaults to `true`), so widgets remain functional
 * even when authoring mistakes are present.
 */
export function evaluateShow(expression: string | undefined, row: unknown, defaultValue = true): boolean {
  if (!expression || !expression.trim()) return defaultValue;
  try {
    const tokens = tokenize(expression);
    const state: ParserState = { index: 0, tokens };
    const result = parseOr(state, row);
    if (state.index < tokens.length) {
      throw new Error("Trailing tokens after expression");
    }
    return Boolean(result);
  } catch (err) {
    if (typeof console !== "undefined") {
      console.warn(`Dashboard widget expression failed: ${(err as Error).message}`);
    }
    return defaultValue;
  }
}
