import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { markPullRequestReadyForReviewMapper } from "./mark_pull_request_ready_for_review";

describe("markPullRequestReadyForReviewMapper", () => {
  it("shows a curated summary of the pull request that was marked ready", () => {
    const details = markPullRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "hello",
          pullNumber: "42",
        },
        outputs: {
          default: [
            {
              data: {
                number: 42,
                title: "Add new feature",
                draft: false,
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
      Title: "Add new feature",
      "Ready for Review": "Yes",
    });
  });

  it("builds the pull request URL from node metadata when the output has none", () => {
    const details = markPullRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "hello",
          pullNumber: "42",
        },
      }),
    );

    expect(details["Pull Request URL"]).toEqual("https://github.com/testhq/hello/pull/42");
    expect(details["Ready for Review"]).toEqual("-");
    expect(details["Title"]).toEqual("-");
  });

  it("reports a pull request that is still a draft as not ready", () => {
    const details = markPullRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { repository: "hello", pullNumber: "42" },
        outputs: { default: [{ data: { number: 42, draft: true } }] },
      }),
    );

    expect(details["Ready for Review"]).toEqual("No");
  });

  it("falls back to the pull request number from the output", () => {
    const details = markPullRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { repository: "hello" },
        outputs: { default: [{ data: { number: 7, draft: false } }] },
      }),
    );

    expect(details["Pull Request"]).toEqual("#7");
    expect(details["Pull Request URL"]).toEqual("https://github.com/testhq/hello/pull/7");
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Mark pull request ready for review",
    componentName: "github.markPullRequestReadyForReview",
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
