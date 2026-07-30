import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getCommitStatusMapper } from "./get_commit_status";

describe("getCommitStatusMapper", () => {
  it("shows the commit's overall status and pipeline", () => {
    const details = getCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commit",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                id: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                short_id: "18f3e63d",
                status: "failed",
                web_url: "https://gitlab.com/g/p/-/commit/18f3e63d",
                last_pipeline: {
                  id: 1024,
                  ref: "main",
                  sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                  status: "failed",
                  web_url: "https://gitlab.com/g/p/-/pipelines/1024",
                },
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Retrieved At": expect.any(String),
      Status: "failed",
      Commit: "18f3e63d",
      Pipeline: "failed",
      Ref: "main",
      URL: "https://gitlab.com/g/p/-/pipelines/1024",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = getCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commit",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                id: "18f3e63d05582537db6d183d9d557be09e1f90c8",
                short_id: "18f3e63d",
                status: "success",
                web_url: "https://gitlab.com/g/p/-/commit/18f3e63d",
                last_pipeline: { id: 1, ref: "main", status: "success", web_url: "https://gitlab.com/x" },
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

  it("falls back to the commit URL when there is no pipeline", () => {
    const details = getCommitStatusMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.commit",
              timestamp: "2026-02-13T18:04:12.000Z",
              data: {
                id: "abc123def456",
                short_id: "abc123de",
                status: "",
                web_url: "https://gitlab.com/g/p/-/commit/abc123",
              },
            },
          ],
        },
      }),
    );

    expect(details["Status"]).toBe("-");
    expect(details["URL"]).toBe("https://gitlab.com/g/p/-/commit/abc123");
  });

  it("handles missing outputs", () => {
    const details = getCommitStatusMapper.getExecutionDetails(
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
    name: "Get Commit Status",
    componentName: "gitlab.getCommitStatus",
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
