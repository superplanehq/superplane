import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRun } from "@/api-client";

import { collectRunRootEventIdsFromPages } from "./useWidgetData";

function run(
  overrides: Partial<CanvasesCanvasRun> & { rootEvent: { id: string; nodeId?: string } },
): CanvasesCanvasRun {
  return {
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  };
}

const PAGES = [
  {
    runs: [
      run({ rootEvent: { id: "evt-passed-a", nodeId: "trigger-a" }, result: "RESULT_PASSED" }),
      run({ rootEvent: { id: "evt-failed-a", nodeId: "trigger-a" }, result: "RESULT_FAILED" }),
      run({ rootEvent: { id: "evt-passed-b", nodeId: "trigger-b" }, result: "RESULT_PASSED" }),
      run({ rootEvent: { id: "evt-failed-b", nodeId: "trigger-b" }, result: "RESULT_FAILED" }),
      run({ rootEvent: { id: "evt-cancelled", nodeId: "trigger-a" }, result: "RESULT_CANCELLED" }),
    ],
  },
];

describe("collectRunRootEventIdsFromPages", () => {
  it("collects the first N root event ids without filters", () => {
    expect(collectRunRootEventIdsFromPages(PAGES, 2)).toEqual(["evt-passed-a", "evt-failed-a"]);
  });

  it("skips non-matching runs when status filters are set so sideload stays on the visible set", () => {
    expect(collectRunRootEventIdsFromPages(PAGES, 10, { statuses: ["failed"] })).toEqual([
      "evt-failed-a",
      "evt-failed-b",
    ]);
  });

  it("caps matching ids at collectLimit (display window), not the unfiltered page size", () => {
    expect(collectRunRootEventIdsFromPages(PAGES, 1, { statuses: ["failed"] })).toEqual(["evt-failed-a"]);
  });

  it("applies trigger filters before counting toward the sideload limit", () => {
    expect(collectRunRootEventIdsFromPages(PAGES, 10, { triggers: ["trigger-b"] })).toEqual([
      "evt-passed-b",
      "evt-failed-b",
    ]);
  });

  it("resolves trigger name references when a resolver is provided", () => {
    const resolve = (reference: string) => (reference === "deploy" ? "trigger-a" : undefined);
    expect(
      collectRunRootEventIdsFromPages(PAGES, 10, { triggers: ["deploy"], statuses: ["cancelled"] }, resolve),
    ).toEqual(["evt-cancelled"]);
  });

  it("returns no ids when every trigger reference fails to resolve", () => {
    expect(collectRunRootEventIdsFromPages(PAGES, 10, { triggers: ["gone"] }, () => undefined)).toEqual([]);
  });
});
