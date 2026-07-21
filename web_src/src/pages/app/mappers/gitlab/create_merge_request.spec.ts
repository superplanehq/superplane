import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createMergeRequestMapper } from "./create_merge_request";

describe("createMergeRequestMapper", () => {
  it("shows created merge request details", () => {
    const details = createMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T08:46:00.000Z",
              data: {
                id: 1,
                iid: 42,
                project_id: 123,
                title: "feat: add login page",
                state: "opened",
                created_at: "2026-02-13T08:46:00.000Z",
                source_branch: "feature/login-page",
                target_branch: "main",
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      "Merge Request": "!42 feat: add login page",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      "Source Branch": "feature/login-page",
      "Target Branch": "main",
      State: "opened",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = createMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T08:46:00.000Z",
              data: {
                id: 1,
                iid: 42,
                title: "feat: add login page",
                state: "opened",
                created_at: "2026-02-13T08:46:00.000Z",
                source_branch: "feature/login-page",
                target_branch: "main",
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Created At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("falls back to the payload timestamp when created_at is missing", () => {
    const details = createMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-13T08:46:00.000Z",
              data: { id: 1, iid: 42, title: "feat: add login page", state: "opened" },
            },
          ],
        },
      }),
    );

    expect(details["Created At"]).not.toBe("-");
    expect(details["Merge Request"]).toBe("!42 feat: add login page");
  });

  it("handles missing outputs", () => {
    const details = createMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create Merge Request",
    componentName: "gitlab.createMergeRequest",
    isCollapsed: false,
    configuration: {
      project: "123",
      sourceBranch: "feature/login-page",
      targetBranch: "main",
      title: "feat: add login page",
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
