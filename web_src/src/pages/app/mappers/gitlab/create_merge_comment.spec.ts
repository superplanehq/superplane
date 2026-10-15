import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { createMergeCommentMapper } from "./create_merge_comment";

describe("createMergeCommentMapper", () => {
  it("shows created comment details", () => {
    const details = createMergeCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "1",
          body: "Automated comment",
        },
        outputs: {
          default: [
            {
              data: {
                id: 302,
                body: "Automated comment",
                author: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      "Created By": "root",
    });
  });

  it("handles missing outputs", () => {
    const details = createMergeCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", mergeRequestIid: "1" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  return buildGitLabDetailsContext(
    {
      name: "Create Merge Request Comment",
      componentName: "gitlab.createMergeComment",
    },
    execution,
  );
}
