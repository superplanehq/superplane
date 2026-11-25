export type CodeTokenKind =
  | "plain"
  | "key"
  | "string"
  | "number"
  | "literal"
  | "punctuation"
  | "command"
  | "flag"
  | "variable"
  | "comment";

export interface CodeToken {
  kind: CodeTokenKind;
  text: string;
}

export type CodeLanguage = "json" | "shell";

const JSON_TOKEN = /("(?:\\.|[^"\\])*")(\s*:)?|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|\b(true|false|null)\b|([{}[\],:])/g;

const SHELL_TOKEN =
  /((?<=^|\s)#[^\n]*)|('[^']*')|("(?:\\.|[^"\\])*")|(\$\{?[A-Za-z_][A-Za-z0-9_]*\}?)|((?<=\s)--?[A-Za-z][\w-]*)|\b(export|curl|echo|openssl|xxd)\b/gm;

const SHELL_VARIABLE = /\$\{?[A-Za-z_][A-Za-z0-9_]*\}?/g;

export function highlightCode(code: string, language: CodeLanguage): CodeToken[] {
  return language === "json" ? highlightJson(code) : highlightShell(code);
}

export function highlightJson(code: string): CodeToken[] {
  return scan(code, JSON_TOKEN, (match) => {
    const [, string, colon, number, literal, punctuation] = match;
    if (string !== undefined && colon !== undefined) {
      return [
        { kind: "key", text: string },
        { kind: "punctuation", text: colon },
      ];
    }
    if (string !== undefined) return [{ kind: "string", text: string }];
    if (number !== undefined) return [{ kind: "number", text: number }];
    if (literal !== undefined) return [{ kind: "literal", text: literal }];
    return [{ kind: "punctuation", text: punctuation }];
  });
}

export function highlightShell(code: string): CodeToken[] {
  return scan(code, SHELL_TOKEN, (match) => {
    const [, comment, single, double, variable, flag, command] = match;
    if (comment !== undefined) return [{ kind: "comment", text: comment }];
    if (single !== undefined) return singleQuoted(single);
    if (double !== undefined) return doubleQuoted(double);
    if (variable !== undefined) return [{ kind: "variable", text: variable }];
    if (flag !== undefined) return [{ kind: "flag", text: flag }];
    return [{ kind: "command", text: command }];
  });
}

/** A single-quoted JSON payload keeps its JSON colors. */
function singleQuoted(text: string): CodeToken[] {
  const inner = text.slice(1, -1);
  if (!/^\s*[{[]/.test(inner)) return [{ kind: "string", text }];
  return [{ kind: "string", text: "'" }, ...highlightJson(inner), { kind: "string", text: "'" }];
}

function doubleQuoted(text: string): CodeToken[] {
  return scan(text, SHELL_VARIABLE, (match) => [{ kind: "variable", text: match[0] }], "string");
}

function scan(
  code: string,
  pattern: RegExp,
  tokensFor: (match: RegExpMatchArray) => CodeToken[],
  gapKind: CodeTokenKind = "plain",
): CodeToken[] {
  const tokens: CodeToken[] = [];
  let last = 0;
  for (const match of code.matchAll(pattern)) {
    const index = match.index ?? 0;
    if (index > last) tokens.push({ kind: gapKind, text: code.slice(last, index) });
    tokens.push(...tokensFor(match));
    last = index + match[0].length;
  }
  if (last < code.length) tokens.push({ kind: gapKind, text: code.slice(last) });
  return tokens;
}
