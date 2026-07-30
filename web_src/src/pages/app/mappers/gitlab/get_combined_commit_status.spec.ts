import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getCombinedCommitStatusMapper } from "./get_combined_commit_status";

describe("getCombinedCommitStatusMapper", () => {
  it("summarizes the combined commit status", () => {
    const details = getCombinedCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.combinedCommitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                state: "failed",
                sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                total_count: 2,
                statuses: [
                  { id: 93, sha: "18f3e63d05582537", status: "success", name: "ci/superplane" },
                  { id: 92, sha: "18f3e63d05582537", status: "failed", name: "lint" },
                ],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Retrieved At": expect.any(String),
      State: "failed",
      "Total statuses": "2",
      Successful: "1",
      Failed: "1",
      SHA: "18f3e63d",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = getCombinedCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.combinedCommitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                state: "success",
                sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                total_count: 1,
                statuses: [{ id: 93, sha: "18f3e63d05582537", status: "success", name: "ci" }],
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Retrieved At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("handles an empty combined status", () => {
    const details = getCombinedCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.combinedCommitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: { state: "", sha: "main", total_count: 0, statuses: [] },
            },
          ],
        },
      }),
    );

    expect(details["State"]).toBe("-");
    expect(details["Total statuses"]).toBe("0");
    expect(details["Successful"]).toBe("0");
  });

  it("handles missing outputs", () => {
    const details = getCombinedCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Combined Commit Status",
    componentName: "gitlab.getCombinedCommitStatus",
    isCollapsed: false,
    configuration: {
      project: "123",
      ref: "main",
    },
    metadata: {
      project: {
        id: 123,
        name: "felixgateru/hello-world",
        url: "https://gitlab.com/felixgateru/hello-world",
      },
    },
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      ...execution,
    },
  };
}
