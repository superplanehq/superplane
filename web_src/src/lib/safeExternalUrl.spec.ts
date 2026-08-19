import { describe, expect, it } from "vitest";

import { safeExternalUrl } from "./safeExternalUrl";

describe("safeExternalUrl", () => {
  it("returns the normalized URL for http(s) links", () => {
    expect(safeExternalUrl("https://github.com/example/repo/pull/1")).toBe("https://github.com/example/repo/pull/1");
    expect(safeExternalUrl("http://example.com/path?q=1")).toBe("http://example.com/path?q=1");
  });

  it("trims surrounding whitespace before validating", () => {
    expect(safeExternalUrl("  https://example.com/x  ")).toBe("https://example.com/x");
  });

  it("rejects javascript: and other script-executing schemes", () => {
    expect(safeExternalUrl("javascript:alert(1)")).toBeNull();
    expect(safeExternalUrl("JAVASCRIPT:alert(1)")).toBeNull();
    expect(safeExternalUrl("vbscript:msgbox(1)")).toBeNull();
  });

  it("rejects data:, file:, mailto:, tel:, and other non-http schemes", () => {
    expect(safeExternalUrl("data:text/html,<script>alert(1)</script>")).toBeNull();
    expect(safeExternalUrl("file:///etc/passwd")).toBeNull();
    expect(safeExternalUrl("mailto:someone@example.com")).toBeNull();
    expect(safeExternalUrl("tel:+15551234567")).toBeNull();
  });

  it("rejects protocol-relative and backslash-relative URLs", () => {
    expect(safeExternalUrl("//evil.example/pull/1")).toBeNull();
    expect(safeExternalUrl("\\\\evil.example\\pull\\1")).toBeNull();
  });

  it("rejects garbage input", () => {
    expect(safeExternalUrl("")).toBeNull();
    expect(safeExternalUrl("   ")).toBeNull();
    expect(safeExternalUrl(null)).toBeNull();
    expect(safeExternalUrl(undefined)).toBeNull();
    expect(safeExternalUrl("not a url at all")).toBeNull();
    expect(safeExternalUrl("http://")).toBeNull();
  });
});
