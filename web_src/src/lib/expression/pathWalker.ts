// Safe path walker shared by expression dialect adapters (expr-lang and
// widget CEL). Adapters only supply their dialect-specific rewrite hook.

export type IndexableValue = Record<string, unknown> | null | undefined;

export type PathToken = { t: "dot" } | { t: "ident"; v: string } | { t: "key"; v: string };

const IDENT_RE = /^[$A-Za-z_][$A-Za-z0-9_]*/;

const STOP_CHARS = new Set<string>([
  "(",
  ")",
  ",",
  ";",
  ":",
  "?",
  "+",
  "-",
  "*",
  "/",
  "%",
  "|",
  "&",
  "!",
  "=",
  "<",
  ">",
  "\n",
  "\r",
  "\t",
  " ",
]);

function isEscapedAt(input: string, idx: number): boolean {
  let backslashes = 0;
  for (let j = idx - 1; j >= 0 && input[j] === "\\"; j--) backslashes++;
  return backslashes % 2 === 1;
}

export function extractTailPathExpression(expr: string): string {
  const s = expr.trim();
  let bracketDepth = 0;
  let parenDepth = 0;
  let inSingle = false;
  let inDouble = false;

  for (let i = s.length - 1; i >= 0; i--) {
    const ch = s[i];

    if (!inDouble && ch === "'" && !isEscapedAt(s, i)) inSingle = !inSingle;
    else if (!inSingle && ch === '"' && !isEscapedAt(s, i)) inDouble = !inDouble;
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
      if (parenDepth === 0) return s.slice(i + 1).trim();
      parenDepth = Math.max(0, parenDepth - 1);
      continue;
    }

    if (bracketDepth > 0 || parenDepth > 0) continue;
    if (STOP_CHARS.has(ch)) return s.slice(i + 1).trim();
  }

  return s;
}

export function stripWhitespaceOutsideStrings(input: string): string {
  let out = "";
  let inSingle = false;
  let inDouble = false;

  for (let i = 0; i < input.length; i++) {
    const ch = input[i];
    if (!inDouble && ch === "'" && !isEscapedAt(input, i)) inSingle = !inSingle;
    else if (!inSingle && ch === '"' && !isEscapedAt(input, i)) inDouble = !inDouble;

    if (!inSingle && !inDouble && /\s/u.test(ch)) continue;
    out += ch;
  }

  return out;
}

export function tokenizePath(expr: string): PathToken[] | null {
  const tokens: PathToken[] = [];
  let i = 0;

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
        tokens.push({ t: "key", v: String(quotedMatch[2] ?? "").replace(/\\(["'\\])/g, "$1") });
        i += quotedMatch[0].length;
        continue;
      }

      const numberMatch = rest.match(/^\[\s*(\d+)\s*\]/);
      if (numberMatch) {
        tokens.push({ t: "key", v: numberMatch[1] });
        i += numberMatch[0].length;
        continue;
      }

      return null;
    }

    const identMatch = rest.match(IDENT_RE);
    if (identMatch) {
      tokens.push({ t: "ident", v: identMatch[0] });
      i += identMatch[0].length;
      continue;
    }

    return null;
  }

  return tokens;
}

export function walkTokens(tokens: PathToken[], root: unknown): unknown {
  let cur: unknown = root;
  let pos = 0;

  while (pos < tokens.length) {
    const tok = tokens[pos];

    if (tok.t === "dot") {
      pos += 1;
      const next = tokens[pos];
      if (!next) return cur;
      if (next.t !== "ident") return undefined;
      try {
        cur = (cur as IndexableValue)?.[next.v];
      } catch {
        return undefined;
      }
      pos += 1;
      continue;
    }

    if (tok.t === "key") {
      try {
        cur = (cur as IndexableValue)?.[tok.v];
      } catch {
        return undefined;
      }
      pos += 1;
      continue;
    }

    return undefined;
  }

  return cur;
}

export interface SuggestionResolver {
  // Rewrite the whitespace-stripped tail before tokenizing. Return `null` to
  // reject the expression outright (e.g. malformed shorthand).
  rewrite?: (stripped: string) => string | null;
  // Resolve the leading identifier against globals; return the value the
  // rest of the token stream should walk into.
  resolveRoot: (ident: string, globals: Record<string, unknown> | null | undefined) => unknown;
}

export function resolveSuggestionPath(
  expression: string,
  globals: Record<string, unknown> | null | undefined,
  resolver: SuggestionResolver,
): unknown {
  const tail = extractTailPathExpression(expression);
  if (!tail) return undefined;
  const stripped = stripWhitespaceOutsideStrings(tail);
  const rewritten = resolver.rewrite ? resolver.rewrite(stripped) : stripped;
  if (rewritten === null) return undefined;
  const tokens = tokenizePath(rewritten);
  if (!tokens || tokens[0]?.t !== "ident") return undefined;
  return walkTokens(tokens.slice(1), resolver.resolveRoot(tokens[0].v, globals));
}
