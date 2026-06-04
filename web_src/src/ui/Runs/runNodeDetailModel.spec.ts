import { describe, expect, it } from "vitest";
import { getAdjacentRunNodeId, isRunNodeDetailTabAvailable, resolveRunNodeDetailTab } from "./runNodeDetailModel";

const allTabs = {
  hasDetailsSection: true,
  hasPayload: true,
  hasConfig: true,
};

describe("resolveRunNodeDetailTab", () => {
  it("returns the preferred tab when it is available", () => {
    expect(resolveRunNodeDetailTab("payload", allTabs)).toBe("payload");
  });

  it("falls back when the preferred tab is unavailable", () => {
    expect(
      resolveRunNodeDetailTab("payload", {
        hasDetailsSection: true,
        hasPayload: false,
        hasConfig: false,
      }),
    ).toBe("details");
  });

  it("checks tab availability", () => {
    expect(isRunNodeDetailTabAvailable("configuration", allTabs)).toBe(true);
    expect(
      isRunNodeDetailTabAvailable("payload", {
        hasDetailsSection: true,
        hasPayload: false,
        hasConfig: false,
      }),
    ).toBe(false);
  });
});

describe("getAdjacentRunNodeId", () => {
  const chain = ["trigger", "node-a", "node-b"];

  it("returns the previous node id", () => {
    expect(getAdjacentRunNodeId(chain, "node-b", "prev")).toBe("node-a");
  });

  it("returns the next node id", () => {
    expect(getAdjacentRunNodeId(chain, "node-a", "next")).toBe("node-b");
  });

  it("returns null at the ends of the chain", () => {
    expect(getAdjacentRunNodeId(chain, "trigger", "prev")).toBeNull();
    expect(getAdjacentRunNodeId(chain, "node-b", "next")).toBeNull();
  });
});
