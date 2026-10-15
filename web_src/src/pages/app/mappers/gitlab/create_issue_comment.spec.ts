import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { createIssueCommentMapper } from "./create_issue_comment";

describe("createIssueCommentMapper", () => {
  it("shows created comment details", () => {
    const details = createIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "1",
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
    const details = createIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  return buildGitLabDetailsContext(
    {
      name: "Create Issue Comment",
      componentName: "gitlab.createIssueComment",
    },
    execution,
  );
}
