import { describe, expect, it } from "vitest";
import { resolveFactoryNodeCardTitles } from "./resolveFactoryNodeCardTitles";

describe("resolveFactoryNodeCardTitles", () => {
  it("uses component label as title and node name as subtitle", () => {
    expect(
      resolveFactoryNodeCardTitles({
        componentLabel: "Create Pull Request",
        nodeName: "Open draft PR",
        fallbackTitle: "fallback",
      }),
    ).toEqual({
      title: "Create Pull Request",
      subtitle: "Open draft PR",
    });
  });

  it("falls back to title when component label missing", () => {
    expect(
      resolveFactoryNodeCardTitles({
        nodeName: "My node",
        fallbackTitle: "Unnamed component",
      }),
    ).toEqual({
      title: "Unnamed component",
      subtitle: "My node",
    });
  });

  it("omits subtitle when node name empty", () => {
    expect(
      resolveFactoryNodeCardTitles({
        componentLabel: "Agent",
        nodeName: "  ",
        fallbackTitle: "Agent",
      }),
    ).toEqual({
      title: "Agent",
      subtitle: null,
    });
  });
});
