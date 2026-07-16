import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRun } from "@/api-client";

import { buildRunsDataSourceRows } from "./runsDataSourceResult";

function failedRun(id: string): CanvasesCanvasRun {
  return {
    id,
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    rootEvent: { id: `event-${id}` },
  };
}

describe("buildRunsDataSourceRows", () => {
  it("caps filtered rows after matching them", () => {
    const rows = buildRunsDataSourceRows({
      dataSource: { kind: "runs", limit: 2, statuses: ["failed"] },
      pages: [{ runs: [failedRun("run-1"), failedRun("run-2"), failedRun("run-3")] }],
      ctx: undefined,
      collectLimit: 3,
      executionsByRootEventId: new Map(),
      runsFilters: { statuses: ["failed"], triggers: undefined },
      resultLimit: 2,
    });

    expect(rows.map((row) => (row as CanvasesCanvasRun).id)).toEqual(["run-1", "run-2"]);
  });
});
