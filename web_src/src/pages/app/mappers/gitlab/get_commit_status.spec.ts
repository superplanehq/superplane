import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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

    expect(Object.keys(details)[0]).toBe("Retrieved At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(details).toEqual({
      "Retrieved At": expect.any(String),
      Status: "failed",
      Commit: "18f3e63d",
      Pipeline: "failed",
      Ref: "main",
      URL: "https://gitlab.com/g/p/-/pipelines/1024",
    });
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
  return buildGitLabDetailsContext(
    {
      name: "Get Commit Status",
      componentName: "gitlab.getCommitStatus",
      configuration: {
        project: "123",
        ref: "main",
      },
    },
    execution,
  );
}
