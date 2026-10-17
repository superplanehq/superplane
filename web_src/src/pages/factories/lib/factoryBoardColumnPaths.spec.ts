import { describe, expect, it } from "bun:test";

import { boardColumnKeyFromSearch, withBoardColumn } from "./factoryBoardColumnPaths";
import { factoryHomePath, workOrderDetailPath } from "./factoryPagePaths";

describe("withBoardColumn", () => {
  it("adds the column key to a line board path", () => {
    expect(withBoardColumn(factoryHomePath("org-1", "sp", "line-plan"), "phase-0")).toBe(
      "/org-1/workspaces/sp/lines/line-plan?column=phase-0",
    );
  });

  it("adds the column key after an existing query", () => {
    expect(withBoardColumn(workOrderDetailPath("org-1", "sp", "42", "line-plan"), "done")).toBe(
      "/org-1/workspaces/sp/task/42?lineId=line-plan&column=done",
    );
  });

  it("returns the path unchanged when the column key is missing or blank", () => {
    expect(withBoardColumn("/org-1/workspaces/sp/lines/line-plan", null)).toBe("/org-1/workspaces/sp/lines/line-plan");
    expect(withBoardColumn("/org-1/workspaces/sp/lines/line-plan", "  ")).toBe("/org-1/workspaces/sp/lines/line-plan");
  });
});

describe("boardColumnKeyFromSearch", () => {
  it("reads the column key from the search string", () => {
    expect(boardColumnKeyFromSearch("?column=verify")).toBe("verify");
    expect(boardColumnKeyFromSearch("intake=1&column=done")).toBe("done");
  });

  it("returns null when the column is missing or blank", () => {
    expect(boardColumnKeyFromSearch("")).toBeNull();
    expect(boardColumnKeyFromSearch("?column=")).toBeNull();
    expect(boardColumnKeyFromSearch("?column=%20%20")).toBeNull();
  });
});
