import { describe, expect, it } from "vitest";

import { hrefFromInput } from "./workOrderDescriptionHref";

describe("hrefFromInput", () => {
  it("returns an empty string for javascript and data URLs", () => {
    expect(hrefFromInput("javascript:alert(1)")).toBe("");
    expect(hrefFromInput("JAVASCRIPT:alert(1)")).toBe("");
    expect(hrefFromInput("data:text/html,<script>alert(1)</script>")).toBe("");
  });

  it("keeps http(s) URLs and adds https when the scheme is missing", () => {
    expect(hrefFromInput("https://example.com/docs")).toBe("https://example.com/docs");
    expect(hrefFromInput("example.com/docs")).toBe("https://example.com/docs");
  });

  it("returns an empty string for blank input", () => {
    expect(hrefFromInput("   ")).toBe("");
  });
});
