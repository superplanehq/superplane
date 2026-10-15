import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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

    expect(Object.keys(details)[0]).toBe("Published At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(details).toEqual({
      "Published At": expect.any(String),
      Commit: "18f3e63d",
      State: "success",
      Context: "ci/superplane",
      "Status URL": "https://gitlab.com/my-group/my-project/-/pipelines/1024",
      Description: "Build passed",
    });
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

    expect(details["Published At"]).toBe(new Date("2026-02-13T18:04:12.000Z").toLocaleString());
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
  return buildGitLabDetailsContext(
    {
      name: "Publish Commit Status",
      componentName: "gitlab.publishCommitStatus",
      configuration: {
        project: "123",
        sha: "18f3e63d05582537db6d183d9d557be09e1f90c8",
        state: "success",
      },
    },
    execution,
  );
}
