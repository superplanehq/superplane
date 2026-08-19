import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { publishCommitStatusMapper } from "./publish_commit_status";

describe("publishCommitStatusMapper", () => {
  it("shows published commit status details", () => {
    const details = publishCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                id: 93,
                sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                ref: "main",
                status: "success",
                name: "ci/superplane",
                target_url: "https://gitlab.com/my-group/my-project/-/pipelines/1024",
                description: "Build passed",
                created_at: "2026-02-13T18:00:00.000Z",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Published At": expect.any(String),
      Commit: "18f3e63d",
      State: "success",
      Context: "ci/superplane",
      "Status URL": "https://gitlab.com/my-group/my-project/-/pipelines/1024",
      Description: "Build passed",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = publishCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                id: 93,
                sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                ref: "main",
                status: "success",
                name: "ci/superplane",
                target_url: "https://gitlab.com/my-group/my-project/-/pipelines/1024",
                description: "Build passed",
                created_at: "2026-02-13T18:00:00.000Z",
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Published At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("falls back to the payload timestamp when created_at is missing", () => {
    const details = publishCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commitStatus",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: { id: 93, sha: "18f3e63d05582537", status: "pending" },
            },
          ],
        },
      }),
    );

    expect(details["Published At"]).not.toBe("-");
    expect(details["State"]).toBe("pending");
  });

  it("handles missing outputs", () => {
    const details = publishCommitStatusMapper.getExecutionDetails(
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
    name: "Publish Commit Status",
    componentName: "gitlab.publishCommitStatus",
    isCollapsed: false,
    configuration: {
      project: "123",
      sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
      state: "success",
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
