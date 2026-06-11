import { describe, expect, it } from "vitest";
import {
  buildSidebarRunIds,
  canNavigateToOlderRun,
  getAdjacentSidebarRunId,
  getRunSidebarNavigation,
  isAtOlderRunPaginationBoundary,
} from "./runsSidebarNavigation";

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

  it("detects the older pagination boundary", () => {
    const runIds = buildSidebarRunIds(orderedRuns);
    expect(isAtOlderRunPaginationBoundary(runIds, "run-older")).toBe(true);
    expect(isAtOlderRunPaginationBoundary(runIds, "run-newer")).toBe(false);
  });

  it("allows older navigation at the pagination boundary when more runs exist", () => {
    const runIds = buildSidebarRunIds(orderedRuns);
    expect(canNavigateToOlderRun(runIds, "run-older", false)).toBe(false);
    expect(canNavigateToOlderRun(runIds, "run-older", true)).toBe(true);
    expect(canNavigateToOlderRun(runIds, "run-newer", true)).toBe(true);
    expect(canNavigateToOlderRun(runIds, "run-older", true, false)).toBe(false);
  });

  it("includes pagination in sidebar navigation state", () => {
    expect(getRunSidebarNavigation(orderedRuns, "run-older", { hasNextPage: false })).toMatchObject({
      olderRunId: null,
      canNavigateOlder: false,
      atOlderPaginationBoundary: true,
    });
    expect(getRunSidebarNavigation(orderedRuns, "run-older", { hasNextPage: true })).toMatchObject({
      olderRunId: null,
      canNavigateOlder: true,
      atOlderPaginationBoundary: true,
    });
    expect(
      getRunSidebarNavigation(orderedRuns, "run-older", { hasNextPage: true, hasActiveFilters: true }),
    ).toMatchObject({
      olderRunId: null,
      canNavigateOlder: false,
      atOlderPaginationBoundary: true,
    });
  });
});
