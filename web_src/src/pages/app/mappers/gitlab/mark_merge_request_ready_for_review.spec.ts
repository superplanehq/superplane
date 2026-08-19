import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { markMergeRequestReadyForReviewMapper } from "./mark_merge_request_ready_for_review";

describe("markMergeRequestReadyForReviewMapper", () => {
  it("shows the merge request as ready for review", () => {
    const details = markMergeRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "42",
        },
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T09:02:11.000Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                state: "opened",
                draft: false,
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": expect.any(String),
      "Merge Request": "!42 feat: add login page",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      "Ready for Review": "Yes",
    });
  });

  it("handles missing outputs", () => {
    const details = markMergeRequestReadyForReviewMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", mergeRequestIid: "42" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and merge request IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = markMergeRequestReadyForReviewMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.markMergeRequestReadyForReview",
        label: "Mark Merge Request Ready for Review",
        description: "",
        icon: "gitlab",
        color: "orange",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    });

    expect(props.metadata).toEqual([
      { icon: "book", label: "felixgateru/hello-world" },
      { icon: "git-merge", label: "!42" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Mark Merge Request Ready for Review",
    componentName: "gitlab.markMergeRequestReadyForReview",
    isCollapsed: false,
    configuration: {
      project: "123",
      mergeRequestIid: "42",
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
