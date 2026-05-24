/**
 * Client-side expression evaluator compatible with expr-lang syntax.
 * Supports path resolution, function calls, method chaining, and operators.
 */

// ============================================================================
// TYPES
// ============================================================================

type TokenType =
  | "IDENTIFIER"
  | "NUMBER"
  | "STRING"
  | "BOOLEAN"
  | "NULL"
  | "OPERATOR"
  | "LPAREN"
  | "RPAREN"
  | "LBRACE"
  | "RBRACE"
  | "LBRACKET"
  | "RBRACKET"
  | "DOT"
  | "COMMA"
  | "QUESTION"
  | "COLON"
  | "EOF";

interface Token {
  type: TokenType;
  value: string;
  pos: number;
}

type ASTNode =
  | { type: "Literal"; value: unknown }
  | { type: "ObjectLiteral"; properties: { key: string; value: ASTNode }[] }
  | { type: "Identifier"; name: string }
  | { type: "MemberAccess"; object: ASTNode; property: string; optional: boolean }
  | { type: "IndexAccess"; object: ASTNode; index: ASTNode }
  | { type: "FunctionCall"; callee: ASTNode; args: ASTNode[] }
  | { type: "MethodCall"; object: ASTNode; method: string; args: ASTNode[] }
  | { type: "UnaryOp"; operator: string; operand: ASTNode }
  | { type: "BinaryOp"; operator: string; left: ASTNode; right: ASTNode }
  | { type: "Ternary"; condition: ASTNode; consequent: ASTNode; alternate: ASTNode }
  | { type: "NilCoalesce"; left: ASTNode; right: ASTNode };

// ============================================================================
// TOKENIZER
// ============================================================================

const OPERATORS = ["??", "||", "&&", "==", "!=", ">=", "<=", "?.", ">", "<", "+", "-", "*", "/", "%", "!"];

function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let pos = 0;

  while (pos < input.length) {
    const char = input[pos];

    // Skip whitespace
    if (/\s/.test(char)) {
      pos++;
      continue;
    }

    // String literals (double or single quotes)
    if (char === '"' || char === "'") {
      const quote = char;
      const start = pos;
      pos++;
      let value = "";
      while (pos < input.length && input[pos] !== quote) {
        if (input[pos] === "\\" && pos + 1 < input.length) {
          pos++;
          const escaped = input[pos];
          if (escaped === "n") value += "\n";
          else if (escaped === "t") value += "\t";
          else if (escaped === "r") value += "\r";
          else value += escaped;
        } else {
          value += input[pos];
        }
        pos++;
      }
      pos++; // Skip closing quote
      tokens.push({ type: "STRING", value, pos: start });
      continue;
    }

    // Numbers
    if (/\d/.test(char) || (char === "." && pos + 1 < input.length && /\d/.test(input[pos + 1]))) {
      const start = pos;
      let numStr = "";
      while (pos < input.length && /[\d.]/.test(input[pos])) {
        numStr += input[pos];
        pos++;
      }
      tokens.push({ type: "NUMBER", value: numStr, pos: start });
      continue;
    }

    // Identifiers and keywords
    if (/[a-zA-Z_$]/.test(char)) {
      const start = pos;
      let ident = "";
      while (pos < input.length && /[a-zA-Z0-9_$]/.test(input[pos])) {
        ident += input[pos];
        pos++;
      }
      if (ident === "true" || ident === "false") {
        tokens.push({ type: "BOOLEAN", value: ident, pos: start });
      } else if (ident === "nil" || ident === "null") {
        tokens.push({ type: "NULL", value: ident, pos: start });
      } else if (ident === "and") {
        tokens.push({ type: "OPERATOR", value: "&&", pos: start });
      } else if (ident === "or") {
        tokens.push({ type: "OPERATOR", value: "||", pos: start });
      } else if (ident === "not") {
        tokens.push({ type: "OPERATOR", value: "!", pos: start });
      } else {
        tokens.push({ type: "IDENTIFIER", value: ident, pos: start });
      }
      continue;
    }

    // Multi-character operators
    let foundOp = false;
    for (const op of OPERATORS) {
      if (input.slice(pos, pos + op.length) === op) {
        tokens.push({ type: "OPERATOR", value: op, pos });
        pos += op.length;
        foundOp = true;
        break;
      }
    }
    if (foundOp) continue;

    // Single character tokens
    if (char === "(") {
      tokens.push({ type: "LPAREN", value: "(", pos });
      pos++;
      continue;
    }
    if (char === ")") {
      tokens.push({ type: "RPAREN", value: ")", pos });
      pos++;
      continue;
    }
    if (char === "[") {
      tokens.push({ type: "LBRACKET", value: "[", pos });
      pos++;
      continue;
    }
    if (char === "{") {
      tokens.push({ type: "LBRACE", value: "{", pos });
      pos++;
      continue;
    }
    if (char === "}") {
      tokens.push({ type: "RBRACE", value: "}", pos });
      pos++;
      continue;
    }
    if (char === "]") {
      tokens.push({ type: "RBRACKET", value: "]", pos });
      pos++;
      continue;
    }
    if (char === ".") {
      tokens.push({ type: "DOT", value: ".", pos });
      pos++;
      continue;
    }
    if (char === ",") {
      tokens.push({ type: "COMMA", value: ",", pos });
      pos++;
      continue;
    }
    if (char === "?") {
      tokens.push({ type: "QUESTION", value: "?", pos });
      pos++;
      continue;
    }
    if (char === ":") {
      tokens.push({ type: "COLON", value: ":", pos });
      pos++;
      continue;
    }

    // Unknown character - skip
    pos++;
  }

  tokens.push({ type: "EOF", value: "", pos });
  return tokens;
}

// ============================================================================
// PARSER
// ============================================================================

class Parser {
  private tokens: Token[];
  private pos: number;

  constructor(tokens: Token[]) {
    this.tokens = tokens;
    this.pos = 0;
  }

  private current(): Token {
    return this.tokens[this.pos] || { type: "EOF", value: "", pos: -1 };
  }

  private advance(): Token {
    const token = this.current();
    this.pos++;
    return token;
  }

  private expect(type: TokenType): Token {
    const token = this.current();
    if (token.type !== type) {
      throw new Error(`Expected ${type} but got ${token.type}`);
    }
    return this.advance();
  }

  parse(): ASTNode {
    return this.parseExpression();
  }

  private parseExpression(): ASTNode {
    return this.parseTernary();
  }

  private parseTernary(): ASTNode {
    let node = this.parseNilCoalesce();

    if (this.current().type === "QUESTION") {
      this.advance(); // consume ?
      const consequent = this.parseExpression();
      this.expect("COLON");
      const alternate = this.parseExpression();
      node = { type: "Ternary", condition: node, consequent, alternate };
    }

    return node;
  }

  private parseNilCoalesce(): ASTNode {
    let node = this.parseOr();

    while (this.current().type === "OPERATOR" && this.current().value === "??") {
      this.advance();
      const right = this.parseOr();
      node = { type: "NilCoalesce", left: node, right };
    }

    return node;
  }

  private parseOr(): ASTNode {
    let node = this.parseAnd();

    while (this.current().type === "OPERATOR" && this.current().value === "||") {
      const op = this.advance().value;
      const right = this.parseAnd();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseAnd(): ASTNode {
    let node = this.parseEquality();

    while (this.current().type === "OPERATOR" && this.current().value === "&&") {
      const op = this.advance().value;
      const right = this.parseEquality();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseEquality(): ASTNode {
    let node = this.parseComparison();

    while (this.current().type === "OPERATOR" && (this.current().value === "==" || this.current().value === "!=")) {
      const op = this.advance().value;
      const right = this.parseComparison();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseComparison(): ASTNode {
    let node = this.parseAdditive();

    while (
      this.current().type === "OPERATOR" &&
      (this.current().value === ">" ||
        this.current().value === "<" ||
        this.current().value === ">=" ||
        this.current().value === "<=")
    ) {
      const op = this.advance().value;
      const right = this.parseAdditive();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseAdditive(): ASTNode {
    let node = this.parseMultiplicative();

    while (this.current().type === "OPERATOR" && (this.current().value === "+" || this.current().value === "-")) {
      const op = this.advance().value;
      const right = this.parseMultiplicative();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseMultiplicative(): ASTNode {
    let node = this.parseUnary();

    while (
      this.current().type === "OPERATOR" &&
      (this.current().value === "*" || this.current().value === "/" || this.current().value === "%")
    ) {
      const op = this.advance().value;
      const right = this.parseUnary();
      node = { type: "BinaryOp", operator: op, left: node, right };
    }

    return node;
  }

  private parseUnary(): ASTNode {
    if (this.current().type === "OPERATOR" && (this.current().value === "!" || this.current().value === "-")) {
      const op = this.advance().value;
      const operand = this.parseUnary();
      return { type: "UnaryOp", operator: op, operand };
    }

    return this.parsePostfix();
  }

  private parsePostfix(): ASTNode {
    let node = this.parsePrimary();

    while (true) {
      // Optional chaining: ?.
      if (this.current().type === "OPERATOR" && this.current().value === "?.") {
        this.advance();
        const propToken = this.expect("IDENTIFIER");
        // Check for method call
        if (this.current().type === "LPAREN") {
          this.advance();
          const args = this.parseArguments();
          this.expect("RPAREN");
          node = { type: "MethodCall", object: node, method: propToken.value, args };
        } else {
          node = { type: "MemberAccess", object: node, property: propToken.value, optional: true };
        }
        continue;
      }

      // Dot access: .property or .method()
      if (this.current().type === "DOT") {
        this.advance();
        const propToken = this.expect("IDENTIFIER");
        // Check for method call
        if (this.current().type === "LPAREN") {
          this.advance();
          const args = this.parseArguments();
          this.expect("RPAREN");
          node = { type: "MethodCall", object: node, method: propToken.value, args };
        } else {
          node = { type: "MemberAccess", object: node, property: propToken.value, optional: false };
        }
        continue;
      }

      // Bracket access: [index] or ["key"]
      if (this.current().type === "LBRACKET") {
        this.advance();
        const index = this.parseExpression();
        this.expect("RBRACKET");
        node = { type: "IndexAccess", object: node, index };
        continue;
      }

      // Function call: func(args)
      if (this.current().type === "LPAREN" && node.type === "Identifier") {
        this.advance();
        const args = this.parseArguments();
        this.expect("RPAREN");
        node = { type: "FunctionCall", callee: node, args };
        continue;
      }

      break;
    }

    return node;
  }

  private parseArguments(): ASTNode[] {
    const args: ASTNode[] = [];

    if (this.current().type !== "RPAREN") {
      args.push(this.parseExpression());

      while (this.current().type === "COMMA") {
        this.advance();
        args.push(this.parseExpression());
      }
    }

    return args;
  }

  private parsePrimary(): ASTNode {
    const token = this.current();

    if (token.type === "NUMBER") {
      this.advance();
      const value = token.value.includes(".") ? parseFloat(token.value) : parseInt(token.value, 10);
      return { type: "Literal", value };
    }

    if (token.type === "STRING") {
      this.advance();
      return { type: "Literal", value: token.value };
    }

    if (token.type === "BOOLEAN") {
      this.advance();
      return { type: "Literal", value: token.value === "true" };
    }

    if (token.type === "NULL") {
      this.advance();
      return { type: "Literal", value: null };
    }

    if (token.type === "IDENTIFIER") {
      this.advance();
      return { type: "Identifier", name: token.value };
    }

    if (token.type === "LPAREN") {
      this.advance();
      const expr = this.parseExpression();
      this.expect("RPAREN");
      return expr;
    }

    if (token.type === "LBRACE") {
      this.advance();
      const properties: { key: string; value: ASTNode }[] = [];

      if (this.current().type !== "RBRACE") {
        while (true) {
          const keyToken = this.current();
          let key: string;
          if (keyToken.type === "STRING" || keyToken.type === "IDENTIFIER") {
            key = keyToken.value;
            this.advance();
          } else {
            throw new Error(`Expected object key but got ${keyToken.type}`);
          }

          this.expect("COLON");
          const value = this.parseExpression();
          properties.push({ key, value });

          if (this.current().type !== "COMMA") {
            break;
          }
          this.advance();
        }
      }

      this.expect("RBRACE");
      return { type: "ObjectLiteral", properties };
    }

    throw new Error(`Unexpected token: ${token.type} (${token.value})`);
  }
}

// ============================================================================
// EVALUATOR
// ============================================================================

// Wrapper for Duration that provides expr-lang compatible methods
class ExprDuration {
  constructor(private ms: number) {}

  Hours(): number {
    return this.ms / (1000 * 60 * 60);
  }
  Minutes(): number {
    return this.ms / (1000 * 60);
  }
  Seconds(): number {
    return this.ms / 1000;
  }
  Milliseconds(): number {
    return this.ms;
  }
  Microseconds(): number {
    return this.ms * 1000;
  }
  Nanoseconds(): number {
    return this.ms * 1000000;
  }
  Abs(): ExprDuration {
    return new ExprDuration(Math.abs(this.ms));
  }
  Round(m?: ExprDuration): ExprDuration {
    if (!m || !(m instanceof ExprDuration)) {
      throw new Error("Round() requires a duration argument");
    }
    const ms = m.Milliseconds();
    if (ms <= 0) return this;
    return new ExprDuration(Math.round(this.ms / ms) * ms);
  }
  Truncate(m?: ExprDuration): ExprDuration {
    if (!m || !(m instanceof ExprDuration)) {
      throw new Error("Truncate() requires a duration argument");
    }
    const ms = m.Milliseconds();
    if (ms <= 0) return this;
    return new ExprDuration(Math.floor(this.ms / ms) * ms);
  }
  toString(): string {
    return `${this.ms}ms`;
  }
}

// Wrapper for Date that provides expr-lang compatible methods
class ExprDate {
  constructor(private date: Date) {}

  Year(): number {
    return this.date.getFullYear();
  }
  Month(): number {
    return this.date.getMonth() + 1; // 1-indexed like Go
  }
  Day(): number {
    return this.date.getDate();
  }
  Hour(): number {
    return this.date.getHours();
  }
  Minute(): number {
    return this.date.getMinutes();
  }
  Second(): number {
    return this.date.getSeconds();
  }
  Weekday(): number {
    return this.date.getDay();
  }
  YearDay(): number {
    const start = new Date(this.date.getFullYear(), 0, 0);
    const diff = this.date.getTime() - start.getTime();
    return Math.floor(diff / (1000 * 60 * 60 * 24));
  }
  Unix(): number {
    return Math.floor(this.date.getTime() / 1000);
  }
  UnixMilli(): number {
    return this.date.getTime();
  }
  UnixMicro(): number {
    return this.date.getTime() * 1000;
  }
  UnixNano(): number {
    return this.date.getTime() * 1000000;
  }
  Format(layout?: string): string {
    // Simple format implementation (Go-style layout patterns)
    // Default to ISO-like format if no layout provided
    const fmt = layout ?? "2006-01-02 15:04:05";
    const pad = (n: number, width: number = 2) => String(n).padStart(width, "0");
    return fmt
      .replace("2006", String(this.date.getFullYear()))
      .replace("01", pad(this.date.getMonth() + 1))
      .replace("02", pad(this.date.getDate()))
      .replace("15", pad(this.date.getHours()))
      .replace("04", pad(this.date.getMinutes()))
      .replace("05", pad(this.date.getSeconds()));
  }
  Add(duration?: ExprDuration): ExprDate {
    if (!duration || !(duration instanceof ExprDuration)) {
      throw new Error('Add() requires a duration argument, e.g., Add(duration("1h"))');
    }
    return new ExprDate(new Date(this.date.getTime() + duration.Milliseconds()));
  }
  Sub(other?: ExprDate): ExprDuration {
    if (!other || !(other instanceof ExprDate)) {
      throw new Error("Sub() requires a time argument");
    }
    return new ExprDuration(this.date.getTime() - other.date.getTime());
  }
  Before(other?: ExprDate): boolean {
    if (!other || !(other instanceof ExprDate)) {
      throw new Error("Before() requires a time argument");
    }
    return this.date.getTime() < other.date.getTime();
  }
  After(other?: ExprDate): boolean {
    if (!other || !(other instanceof ExprDate)) {
      throw new Error("After() requires a time argument");
    }
    return this.date.getTime() > other.date.getTime();
  }
  Equal(other?: ExprDate): boolean {
    if (!other || !(other instanceof ExprDate)) {
      throw new Error("Equal() requires a time argument");
    }
    return this.date.getTime() === other.date.getTime();
  }
  In(): ExprDate {
    // Timezone conversion not fully supported in JS - return as-is
    return this;
  }
  UTC(): ExprDate {
    return this; // JS dates are already in UTC internally
  }
  Local(): ExprDate {
    return this;
  }
  IsZero(): boolean {
    return this.date.getTime() === 0;
  }
  Round(duration?: ExprDuration): ExprDate {
    if (!duration || !(duration instanceof ExprDuration)) {
      throw new Error('Round() requires a duration argument, e.g., Round(duration("1h"))');
    }
    const ms = duration.Milliseconds();
    if (ms <= 0) return this;
    const rounded = Math.round(this.date.getTime() / ms) * ms;
    return new ExprDate(new Date(rounded));
  }
  Truncate(duration?: ExprDuration): ExprDate {
    if (!duration || !(duration instanceof ExprDuration)) {
      throw new Error('Truncate() requires a duration argument, e.g., Truncate(duration("1h"))');
    }
    const ms = duration.Milliseconds();
    if (ms <= 0) return this;
    const truncated = Math.floor(this.date.getTime() / ms) * ms;
    return new ExprDate(new Date(truncated));
  }
  toString(): string {
    return this.date.toISOString();
  }
}

// Built-in functions
const BUILTIN_FUNCTIONS: Record<string, (...args: unknown[]) => unknown> = {
  // String functions
  trim: (str: unknown, chars?: unknown) => {
    const s = String(str);
    if (chars === undefined) return s.trim();
    const c = String(chars);
    let result = s;
    while (result.length > 0 && c.includes(result[0])) result = result.slice(1);
    while (result.length > 0 && c.includes(result[result.length - 1])) result = result.slice(0, -1);
    return result;
  },
  trimPrefix: (str: unknown, prefix: unknown) => {
    const s = String(str);
    const p = String(prefix);
    return s.startsWith(p) ? s.slice(p.length) : s;
  },
  trimSuffix: (str: unknown, suffix: unknown) => {
    const s = String(str);
    const suf = String(suffix);
    return s.endsWith(suf) ? s.slice(0, -suf.length) : s;
  },
  upper: (str: unknown) => String(str).toUpperCase(),
  lower: (str: unknown) => String(str).toLowerCase(),
  split: (str: unknown, delim: unknown, n?: unknown) => {
    const s = String(str);
    const d = String(delim);
    if (n !== undefined) return s.split(d, Number(n));
    return s.split(d);
  },
  replace: (str: unknown, old: unknown, newStr: unknown) => {
    return String(str).split(String(old)).join(String(newStr));
  },
  repeat: (str: unknown, n: unknown) => String(str).repeat(Number(n)),
  indexOf: (str: unknown, sub: unknown) => String(str).indexOf(String(sub)),
  lastIndexOf: (str: unknown, sub: unknown) => String(str).lastIndexOf(String(sub)),
  hasPrefix: (str: unknown, prefix: unknown) => String(str).startsWith(String(prefix)),
  hasSuffix: (str: unknown, suffix: unknown) => String(str).endsWith(String(suffix)),
  contains: (str: unknown, sub: unknown) => String(str).includes(String(sub)),

  // Date functions
  now: () => new ExprDate(new Date()),
  date: (str: unknown) => new ExprDate(new Date(String(str))),
  duration: (str: unknown) => {
    // Parse duration string like "1h", "30m", "1h30m", "2s", "500ms"
    const s = String(str);
    let ms = 0;
    const patterns = [
      { regex: /(\d+)h/g, mult: 60 * 60 * 1000 },
      { regex: /(\d+)m(?!s)/g, mult: 60 * 1000 },
      { regex: /(\d+)s(?!$)/g, mult: 1000 },
      { regex: /(\d+)ms/g, mult: 1 },
      { regex: /(\d+)us/g, mult: 0.001 },
      { regex: /(\d+)ns/g, mult: 0.000001 },
    ];
    for (const { regex, mult } of patterns) {
      let match;
      while ((match = regex.exec(s)) !== null) {
        ms += parseInt(match[1], 10) * mult;
      }
    }
    // Handle plain seconds at end (e.g., "30s")
    const plainSeconds = s.match(/(\d+)s$/);
    if (plainSeconds) {
      ms += parseInt(plainSeconds[1], 10) * 1000;
    }
    return new ExprDuration(ms);
  },

  // Number functions
  max: (a: unknown, b: unknown) => Math.max(Number(a), Number(b)),
  min: (a: unknown, b: unknown) => Math.min(Number(a), Number(b)),
  abs: (n: unknown) => Math.abs(Number(n)),
  ceil: (n: unknown) => Math.ceil(Number(n)),
  floor: (n: unknown) => Math.floor(Number(n)),
  round: (n: unknown) => Math.round(Number(n)),

  // Array/Object functions
  len: (v: unknown) => {
    if (typeof v === "string") return v.length;
    if (Array.isArray(v)) return v.length;
    if (v && typeof v === "object") return Object.keys(v).length;
    return 0;
  },
  keys: (obj: unknown) => {
    if (obj && typeof obj === "object" && !Array.isArray(obj)) {
      return Object.keys(obj);
    }
    return [];
  },
  values: (obj: unknown) => {
    if (obj && typeof obj === "object" && !Array.isArray(obj)) {
      return Object.values(obj);
    }
    return [];
  },
  first: (arr: unknown) => {
    if (Array.isArray(arr) && arr.length > 0) return arr[0];
    return null;
  },
  last: (arr: unknown) => {
    if (Array.isArray(arr) && arr.length > 0) return arr[arr.length - 1];
    return null;
  },
  reverse: (arr: unknown) => {
    if (Array.isArray(arr)) return [...arr].reverse();
    return arr;
  },
  sort: (arr: unknown) => {
    if (Array.isArray(arr)) return [...arr].sort();
    return arr;
  },
  uniq: (arr: unknown) => {
    if (Array.isArray(arr)) return [...new Set(arr)];
    return arr;
  },
  flatten: (arr: unknown) => {
    if (Array.isArray(arr)) return arr.flat();
    return arr;
  },
  concat: (...arrays: unknown[]) => {
    return arrays.reduce<unknown[]>((acc, arr) => {
      if (Array.isArray(arr)) return acc.concat(arr);
      return acc;
    }, []);
  },
  join: (arr: unknown, delim?: unknown) => {
    if (Array.isArray(arr)) return arr.join(delim !== undefined ? String(delim) : "");
    return "";
  },
  sum: (arr: unknown) => {
    if (Array.isArray(arr)) return arr.reduce((a, b) => Number(a) + Number(b), 0);
    return 0;
  },
  mean: (arr: unknown) => {
    if (Array.isArray(arr) && arr.length > 0) {
      const sum = arr.reduce((a, b) => Number(a) + Number(b), 0);
      return sum / arr.length;
    }
    return 0;
  },
  count: (arr: unknown) => {
    if (Array.isArray(arr)) return arr.length;
    return 0;
  },
  take: (arr: unknown, n: unknown) => {
    if (Array.isArray(arr)) return arr.slice(0, Number(n));
    return arr;
  },

  // Type conversion functions
  string: (v: unknown) => {
    if (v === null || v === undefined) return "";
    return String(v);
  },
  int: (v: unknown) => {
    const n = parseInt(String(v), 10);
    return isNaN(n) ? 0 : n;
  },
  float: (v: unknown) => {
    const n = parseFloat(String(v));
    return isNaN(n) ? 0 : n;
  },
  type: (v: unknown) => {
    if (v === null) return "nil";
    if (Array.isArray(v)) return "array";
    return typeof v;
  },
  toJSON: (v: unknown) => JSON.stringify(v),
  fromJSON: (v: unknown) => {
    try {
      return JSON.parse(String(v));
    } catch {
      return null;
    }
  },
  toBase64: (v: unknown) => btoa(String(v)),
  fromBase64: (v: unknown) => atob(String(v)),

  // Misc
  get: (obj: unknown, key: unknown) => {
    if (obj && typeof obj === "object") {
      return (obj as Record<string, unknown>)[String(key)] ?? null;
    }
    return null;
  },
};

function evaluate(node: ASTNode, context: Record<string, unknown>): unknown {
  switch (node.type) {
    case "Literal":
      return node.value;

    case "ObjectLiteral": {
      const result: Record<string, unknown> = {};
      for (const property of node.properties) {
        result[property.key] = evaluate(property.value, context);
      }
      return result;
    }

    case "Identifier":
      if (node.name === "$") {
        return context;
      }
      if (node.name === "root") {
        return () => (context.__root as unknown) ?? null;
      }
      if (node.name === "previous") {
        return (depth?: unknown) => {
          const parsedDepth = depth === undefined ? 1 : Number(depth);
          if (!Number.isInteger(parsedDepth) || parsedDepth < 1) {
            return null;
          }
          const previousMap = context.__previousByDepth;
          if (previousMap && typeof previousMap === "object") {
            return (previousMap as Record<string, unknown>)[String(parsedDepth)] ?? null;
          }
          return null;
        };
      }
      if (node.name in BUILTIN_FUNCTIONS) {
        return BUILTIN_FUNCTIONS[node.name];
      }
      return context[node.name];

    case "MemberAccess": {
      const obj = evaluate(node.object, context);
      if (node.optional && (obj === null || obj === undefined)) {
        return null;
      }
      if (obj && typeof obj === "object") {
        return (obj as Record<string, unknown>)[node.property];
      }
      return undefined;
    }

    case "IndexAccess": {
      const obj = evaluate(node.object, context);
      const index = evaluate(node.index, context);
      if (obj && typeof obj === "object") {
        return (obj as Record<string, unknown>)[String(index)];
      }
      if (Array.isArray(obj) && typeof index === "number") {
        return obj[index];
      }
      return undefined;
    }

    case "FunctionCall": {
      const callee = evaluate(node.callee, context);
      if (typeof callee === "function") {
        const args = node.args.map((arg) => evaluate(arg, context));
        return callee(...args);
      }
      throw new Error(`${node.callee} is not a function`);
    }

    case "MethodCall": {
      const obj = evaluate(node.object, context);
      if (obj === null || obj === undefined) {
        return null;
      }

      // Handle ExprDate methods
      if (obj instanceof ExprDate) {
        const method = obj[node.method as keyof ExprDate];
        if (typeof method === "function") {
          const args = node.args.map((arg) => evaluate(arg, context));
          return (method as (...args: unknown[]) => unknown).apply(obj, args);
        }
      }

      // Handle string methods
      if (typeof obj === "string") {
        const strMethods: Record<string, (...args: unknown[]) => unknown> = {
          contains: (sub: unknown) => obj.includes(String(sub)),
          startsWith: (prefix: unknown) => obj.startsWith(String(prefix)),
          endsWith: (suffix: unknown) => obj.endsWith(String(suffix)),
          toUpperCase: () => obj.toUpperCase(),
          toLowerCase: () => obj.toLowerCase(),
        };
        if (node.method in strMethods) {
          const args = node.args.map((arg) => evaluate(arg, context));
          return strMethods[node.method](...args);
        }
      }

      // Handle array methods
      if (Array.isArray(obj)) {
        const arrMethods: Record<string, (...args: unknown[]) => unknown> = {
          length: () => obj.length,
        };
        if (node.method in arrMethods) {
          const args = node.args.map((arg) => evaluate(arg, context));
          return arrMethods[node.method](...args);
        }
      }

      // Generic object method call
      if (typeof obj === "object" && node.method in obj) {
        const method = (obj as Record<string, unknown>)[node.method];
        if (typeof method === "function") {
          const args = node.args.map((arg) => evaluate(arg, context));
          return method.call(obj, ...args);
        }
      }

      throw new Error(`Method ${node.method} not found`);
    }

    case "UnaryOp": {
      const operand = evaluate(node.operand, context);
      switch (node.operator) {
        case "!":
          return !operand;
        case "-":
          return -Number(operand);
        default:
          throw new Error(`Unknown unary operator: ${node.operator}`);
      }
    }

    case "BinaryOp": {
      const left = evaluate(node.left, context);
      const right = evaluate(node.right, context);

      switch (node.operator) {
        case "+":
          if (typeof left === "string" || typeof right === "string") {
            return String(left) + String(right);
          }
          return Number(left) + Number(right);
        case "-":
          return Number(left) - Number(right);
        case "*":
          return Number(left) * Number(right);
        case "/":
          return Number(left) / Number(right);
        case "%":
          return Number(left) % Number(right);
        case "==":
          return left === right;
        case "!=":
          return left !== right;
        case ">":
          return Number(left) > Number(right);
        case "<":
          return Number(left) < Number(right);
        case ">=":
          return Number(left) >= Number(right);
        case "<=":
          return Number(left) <= Number(right);
        case "&&":
          return Boolean(left) && Boolean(right);
        case "||":
          return Boolean(left) || Boolean(right);
        default:
          throw new Error(`Unknown binary operator: ${node.operator}`);
      }
    }

    case "Ternary": {
      const condition = evaluate(node.condition, context);
      return condition ? evaluate(node.consequent, context) : evaluate(node.alternate, context);
    }

    case "NilCoalesce": {
      const left = evaluate(node.left, context);
      return left !== null && left !== undefined ? left : evaluate(node.right, context);
    }

    default:
      throw new Error(`Unknown node type: ${(node as ASTNode).type}`);
  }
}

// ============================================================================
// PUBLIC API
// ============================================================================

/**
 * Evaluates an expression string against a context object.
 * The context is available as $ in the expression.
 *
 * @example
 * evaluateExpr('$["node"].data.name', { node: { data: { name: "test" } } })
 * // Returns: "test"
 *
 * @example
 * evaluateExpr('now().Year()', {})
 * // Returns: 2024 (current year)
 *
 * @example
 * evaluateExpr('upper("hello")', {})
 * // Returns: "HELLO"
 */
export function evaluateExpr(expression: string, context: Record<string, unknown>): unknown {
  const tokens = tokenize(expression);
  const parser = new Parser(tokens);
  const ast = parser.parse();
  return evaluate(ast, context);
}

/**
 * Formats an evaluated result as a string for display.
 */
export function formatExprResult(value: unknown): string {
  if (value === null || value === undefined) {
    return "null";
  }
  if (value instanceof ExprDate) {
    return value.toString();
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  if (Array.isArray(value)) {
    if (value.length <= 3) {
      return `[${value.map((v) => formatExprResult(v)).join(", ")}]`;
    }
    return `[${value.length} items]`;
  }
  if (typeof value === "object") {
    const keys = Object.keys(value);
    if (keys.length <= 3) {
      return `{${keys.join(", ")}}`;
    }
    return `{${keys.length} keys}`;
  }
  return String(value);
}
