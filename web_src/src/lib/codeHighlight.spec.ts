import { describe, expect, it } from "bun:test";

import { highlightJson, highlightShell, type CodeToken } from "./codeHighlight";

function kinds(tokens: CodeToken[], kind: CodeToken["kind"]) {
  return tokens.filter((token) => token.kind === kind).map((token) => token.text);
}

describe("highlightJson", () => {
  it("tells keys from string values and keeps the source text", () => {
    const code = '{\n  "title": "a: b",\n  "count": 2,\n  "open": true\n}';
    const tokens = highlightJson(code);

    expect(tokens.map((token) => token.text).join("")).toBe(code);
    expect(kinds(tokens, "key")).toEqual(['"title"', '"count"', '"open"']);
    expect(kinds(tokens, "string")).toEqual(['"a: b"']);
    expect(kinds(tokens, "number")).toEqual(["2"]);
    expect(kinds(tokens, "literal")).toEqual(["true"]);
  });
});

describe("highlightShell", () => {
  it("colors a single-quoted JSON payload as JSON", () => {
    const code = `export PAYLOAD='{"title":"Checkout fails"}'\ncurl -X POST -H "Authorization: Bearer $TOKEN" --data "$PAYLOAD" https://example.com/hooks#a`;
    const tokens = highlightShell(code);

    expect(tokens.map((token) => token.text).join("")).toBe(code);
    expect(kinds(tokens, "command")).toEqual(["export", "curl"]);
    expect(kinds(tokens, "key")).toEqual(['"title"']);
    expect(kinds(tokens, "flag")).toEqual(["-X", "-H", "--data"]);
    expect(kinds(tokens, "variable")).toEqual(["$TOKEN", "$PAYLOAD"]);
    expect(kinds(tokens, "comment")).toEqual([]);
  });
});
