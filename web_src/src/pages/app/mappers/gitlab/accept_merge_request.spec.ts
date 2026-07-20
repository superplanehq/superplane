import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { acceptMergeRequestMapper } from "./accept_merge_request";

describe("acceptMergeRequestMapper", () => {
  it("shows merged merge request details", () => {
    const details = acceptMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "42",
        },
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T11:16:17.520Z",
              data: {
                id: 1,
                iid: 42,
                project_id: 123,
                title: "feat: add login page",
                state: "merged",
                merged_at: "2026-02-13T11:16:17.520Z",
                source_branch: "feature/login-page",
                target_branch: "main",
                merge_commit_sha: "9999999999999999999999999999999999999999",
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Merged At": expect.any(String),
      "Merge Request": "!42 feat: add login page",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      "Source Branch": "feature/login-page",
      "Target Branch": "main",
      "Merge Commit SHA": "99999999",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = acceptMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T11:16:17.520Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                state: "merged",
                merged_at: "2026-02-13T11:16:17.520Z",
                source_branch: "feature/login-page",
                target_branch: "main",
                merge_commit_sha: "9999999999999999999999999999999999999999",
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Merged At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("falls back to the payload timestamp when merged_at is missing", () => {
    const details = acceptMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T11:16:17.520Z",
              data: { id: 1, iid: 42, title: "feat: add login page", state: "merged" },
            },
          ],
        },
      }),
    );

    expect(details["Merged At"]).not.toBe("-");
    expect(details["Merge Request"]).toBe("!42 feat: add login page");
  });

  it("handles missing outputs", () => {
    const details = acceptMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", mergeRequestIid: "42" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and merge request IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = acceptMergeRequestMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.acceptMergeRequest",
        label: "Accept Merge Request",
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
    name: "Accept Merge Request",
    componentName: "gitlab.acceptMergeRequest",
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
