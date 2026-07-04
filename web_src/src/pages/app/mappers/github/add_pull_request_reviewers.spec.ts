import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { addPullRequestReviewersMapper } from "./add_pull_request_reviewers";

describe("addPullRequestReviewersMapper", () => {
  it("shows a curated reviewer summary", () => {
    const details = addPullRequestReviewersMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "hello",
          pullNumber: "42",
          reviewers: ["octocat"],
          teamReviewers: ["justice-league"],
        },
        outputs: {
          default: [
            {
              data: {
                number: 42,
                title: "Add new feature",
                state: "open",
                html_url: "https://github.com/testhq/hello/pull/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Repository: "hello",
      "Pull Request": "#42",
      "Pull Request URL": "https://github.com/testhq/hello/pull/42",
      Reviewers: "octocat",
      "Team Reviewers": "justice-league",
      Title: "Add new feature",
      State: "open",
    });
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Add pull request reviewers",
    componentName: "github.addPullRequestReviewers",
    isCollapsed: false,
    metadata: {
      repository: {
        id: "123456",
        name: "hello",
        url: "https://github.com/testhq/hello",
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
