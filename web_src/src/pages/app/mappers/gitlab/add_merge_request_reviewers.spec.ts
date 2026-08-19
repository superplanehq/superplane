import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { addMergeRequestReviewersMapper } from "./add_merge_request_reviewers";

describe("addMergeRequestReviewersMapper", () => {
  it("shows updated merge request reviewer details", () => {
    const details = addMergeRequestReviewersMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T09:30:00.000Z",
              data: {
                id: 1,
                iid: 42,
                project_id: 123,
                title: "feat: add login page",
                state: "opened",
                updated_at: "2026-02-13T09:30:00.000Z",
                reviewers: [
                  { id: 30, username: "amorgan", name: "Alex Morgan", state: "active", avatar_url: "", web_url: "" },
                  { id: 31, username: "schen", name: "Sam Chen", state: "active", avatar_url: "", web_url: "" },
                ],
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": expect.any(String),
      "Merge Request": "!42 feat: add login page",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      Reviewers: "@amorgan, @schen",
      State: "opened",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = addMergeRequestReviewersMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T09:30:00.000Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                state: "opened",
                updated_at: "2026-02-13T09:30:00.000Z",
                reviewers: [
                  { id: 30, username: "amorgan", name: "Alex Morgan", state: "active", avatar_url: "", web_url: "" },
                ],
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Updated At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("shows None when there are no reviewers left", () => {
    const details = addMergeRequestReviewersMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T09:30:00.000Z",
              data: { id: 1, iid: 42, title: "feat: add login page", state: "opened", reviewers: [] },
            },
          ],
        },
      }),
    );

    expect(details["Reviewers"]).toBe("None");
  });

  it("handles missing outputs", () => {
    const details = addMergeRequestReviewersMapper.getExecutionDetails(buildDetailsContext({ outputs: {} }));

    expect(details).toEqual({});
  });

  it("shows project and merge request IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = addMergeRequestReviewersMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.addMergeRequestReviewers",
        label: "Add Merge Request Reviewers",
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
    name: "Add Merge Request Reviewers",
    componentName: "gitlab.addMergeRequestReviewers",
    isCollapsed: false,
    configuration: {
      project: "123",
      mergeRequestIid: "42",
      reviewers: ["31"],
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
