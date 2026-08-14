import { describe, expect, it } from "vitest";
import { resolveFactorySidebarHeading } from "./resolveFactorySidebarHeading";

describe("resolveFactorySidebarHeading", () => {
  it("puts component label first and node name second", () => {
    expect(
      resolveFactorySidebarHeading({
        componentLabel: "Create Pull Request",
        nodeName: "open-pr",
      }),
    ).toEqual({ primary: "Create Pull Request", secondary: "open-pr" });
  });

  it("falls back to node name when label missing", () => {
    expect(resolveFactorySidebarHeading({ nodeName: "open-pr" })).toEqual({
      primary: "open-pr",
      secondary: null,
    });
  });

  it("omits secondary when name missing", () => {
    expect(resolveFactorySidebarHeading({ componentLabel: "Create Pull Request" })).toEqual({
      primary: "Create Pull Request",
      secondary: null,
    });
  });
});
