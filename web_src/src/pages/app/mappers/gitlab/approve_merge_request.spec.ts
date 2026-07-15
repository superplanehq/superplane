import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { approveMergeRequestMapper } from "./approve_merge_request";

describe("approveMergeRequestMapper", () => {
  it("shows approval details", () => {
    const details = approveMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          mergeRequestIid: "42",
        },
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequestApproval",
              timestamp: "2026-02-13T10:15:30.000Z",
              data: {
                id: 1,
                iid: 42,
                project_id: 123,
                title: "feat: add login page",
                state: "opened",
                approvals_required: 2,
                approvals_left: 1,
                approved_by: [
                  {
                    user: { id: 1, name: "Administrator", username: "root" },
                    approved_at: "2026-02-13T10:15:30.000Z",
                  },
                ],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Approved At": expect.any(String),
      "Merge Request": "!42 feat: add login page",
      "Merge Request URL": "https://gitlab.com/felixgateru/hello-world/-/merge_requests/42",
      "Approved By": "root",
      "Approvals Required": "2",
      "Approvals Left": "1",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = approveMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequestApproval",
              timestamp: "2026-02-13T10:15:30.000Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                approvals_required: 2,
                approvals_left: 1,
                approved_by: [{ user: { username: "root" } }],
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Approved At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("omits approval counts when no approvals are required", () => {
    const details = approveMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequestApproval",
              timestamp: "2026-02-13T10:15:30.000Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                approvals_required: 0,
                approvals_left: 0,
                approved_by: [{ user: { username: "root" } }],
              },
            },
          ],
        },
      }),
    );

    expect(details["Approvals Required"]).toBeUndefined();
    expect(details["Approvals Left"]).toBeUndefined();
    expect(details["Approved By"]).toBe("root");
  });

  it("handles missing outputs", () => {
    const details = approveMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", mergeRequestIid: "42" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and merge request IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = approveMergeRequestMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.approveMergeRequest",
        label: "Approve Merge Request",
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
      { icon: "git-pull-request", label: "!42" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Approve Merge Request",
    componentName: "gitlab.approveMergeRequest",
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
