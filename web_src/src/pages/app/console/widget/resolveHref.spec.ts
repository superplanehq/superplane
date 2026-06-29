import { describe, expect, it } from "vitest";

import { resolveHref } from "./resolveHref";

describe("resolveHref", () => {
  it("returns an empty string when no template is supplied", () => {
    expect(resolveHref(undefined, { prUrl: "https://example.com/pr/1" })).toBe("");
    expect(resolveHref("", { prUrl: "https://example.com/pr/1" })).toBe("");
  });

  it("resolves a bare {{ field }} expression to the row value", () => {
    expect(resolveHref("{{ prUrl }}", { prUrl: "https://github.com/org/repo/pull/42" })).toBe(
      "https://github.com/org/repo/pull/42",
    );
  });

  it("resolves mixed {{ }} templates with literal text", () => {
    expect(
      resolveHref("https://github.com/{{ org }}/{{ repo }}/pull/{{ prNumber }}", {
        org: "acme",
        repo: "core",
        prNumber: 42,
      }),
    ).toBe("https://github.com/acme/core/pull/42");
  });

  it("supports CEL builtins inside {{ }}", () => {
    expect(resolveHref("https://x/{{ lower(name) }}", { name: "API" })).toBe("https://x/api");
  });

  it("keeps legacy single-brace {field} placeholders working", () => {
    expect(resolveHref("https://x/{service}/{version}", { service: "api", version: "v2" })).toBe("https://x/api/v2");
  });

  it("leaves bare static URLs untouched", () => {
    expect(resolveHref("https://example.com/static", { prUrl: "ignored" })).toBe("https://example.com/static");
  });

  it("renders missing {{ }} fields as empty strings without crashing", () => {
    expect(resolveHref("https://x/{{ missing }}/end", { other: "value" })).toBe("https://x//end");
  });

  it("renders missing legacy {field} placeholders as empty strings", () => {
    expect(resolveHref("https://x/{missing}/end", {})).toBe("https://x//end");
  });
});
