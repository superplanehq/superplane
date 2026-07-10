import { describe, expect, it } from "vitest";

import { consoleBadgeClassName } from "./consoleBadgeStyles";

describe("consoleBadgeClassName", () => {
  it("uses the shared uppercase badge typography", () => {
    expect(consoleBadgeClassName("deployed")).toContain("text-[10px]");
    expect(consoleBadgeClassName("deployed")).toContain("font-semibold");
    expect(consoleBadgeClassName("deployed")).toContain("uppercase");
    expect(consoleBadgeClassName("deployed")).toContain("tracking-wide");
    expect(consoleBadgeClassName("deployed")).toContain("whitespace-nowrap");
  });

  it("maps deployment and run statuses to semantic colors", () => {
    expect(consoleBadgeClassName("deployed")).toContain("bg-emerald-500");
    expect(consoleBadgeClassName("failed")).toContain("bg-red-500");
    expect(consoleBadgeClassName("running")).toContain("bg-blue-500");
  });

  it("falls back to gray for unknown labels", () => {
    expect(consoleBadgeClassName("api")).toContain("bg-gray-500");
  });
});
