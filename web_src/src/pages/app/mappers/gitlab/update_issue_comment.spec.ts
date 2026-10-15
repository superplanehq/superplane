import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { updateIssueCommentMapper } from "./update_issue_comment";

describe("updateIssueCommentMapper", () => {
  it("shows updated comment details", () => {
    const details = updateIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1", commentId: "302", body: "Updated comment" },
        outputs: {
          default: [
            {
              type: "gitlab.updateIssueComment",
              timestamp: "2026-06-11T14:45:00Z",
              data: {
                id: 302,
                body: "Updated comment",
                author: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
                updated_at: "2026-06-11T14:45:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Updated At");
    expect(details).toEqual({
      "Updated At": expect.any(String),
      "Comment ID": "302",
      Author: "root",
    });
  });

  it("handles missing outputs", () => {
    const details = updateIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1", commentId: "302" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  return buildGitLabDetailsContext(
    {
      name: "Update Issue Comment",
      componentName: "gitlab.updateIssueComment",
    },
    execution,
  );
}
