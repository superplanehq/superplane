import { describe, expect, it } from "vitest";
import { buildSidebarRunIds, getAdjacentSidebarRunId } from "./runsSidebarNavigation";

describe("runsSidebarNavigation", () => {
  const orderedRuns = {
    active: [{ run: { id: "run-running" } }],
    rest: [{ run: { id: "run-newer" } }, { run: { id: "run-older" } }],
  };

  it("builds sidebar run ids in display order", () => {
    expect(buildSidebarRunIds(orderedRuns)).toEqual(["run-running", "run-newer", "run-older"]);
  });

  it("returns the newer run when navigating prev", () => {
    const runIds = buildSidebarRunIds(orderedRuns);
    expect(getAdjacentSidebarRunId(runIds, "run-older", "prev")).toBe("run-newer");
  });

  it("returns the older run when navigating next", () => {
    const runIds = buildSidebarRunIds(orderedRuns);
    expect(getAdjacentSidebarRunId(runIds, "run-newer", "next")).toBe("run-older");
  });

  it("returns null at the ends of the list", () => {
    const runIds = buildSidebarRunIds(orderedRuns);
    expect(getAdjacentSidebarRunId(runIds, "run-running", "prev")).toBeNull();
    expect(getAdjacentSidebarRunId(runIds, "run-older", "next")).toBeNull();
  });
});
