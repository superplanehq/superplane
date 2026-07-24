import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updateMergeRequestMapper } from "./update_merge_request";

describe("updateMergeRequestMapper", () => {
  it("shows updated merge request details", () => {
    const details = updateMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-14T09:12:33.000Z",
              data: {
                id: 155,
                iid: 42,
                project_id: 123,
                title: "feat: add login page (updated)",
                state: "opened",
                created_at: "2026-02-13T08:46:00.000Z",
                updated_at: "2026-02-14T09:12:33.000Z",
                source_branch: "feature/login-page",
                target_branch: "develop",
                web_url: "https://gitlab.com/my-group/my-project/-/merge_requests/42",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": expect.any(String),
      "Merge Request": "!42 feat: add login page (updated)",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      "Source Branch": "feature/login-page",
      "Target Branch": "develop",
      State: "opened",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = updateMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-14T09:12:33.000Z",
              data: {
                id: 155,
                iid: 42,
                title: "feat: add login page (updated)",
                state: "opened",
                updated_at: "2026-02-14T09:12:33.000Z",
                source_branch: "feature/login-page",
                target_branch: "develop",
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

  it("falls back to the payload timestamp when updated_at is missing", () => {
    const details = updateMergeRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.mergeRequest",
              timestamp: "2026-02-14T09:12:33.000Z",
              data: { id: 155, iid: 42, title: "feat: add login page (updated)", state: "opened" },
            },
          ],
        },
      }),
    );

    expect(details["Updated At"]).not.toBe("-");
    expect(details["Merge Request"]).toBe("!42 feat: add login page (updated)");
  });

  it("handles missing outputs", () => {
    const details = updateMergeRequestMapper.getExecutionDetails(
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
    name: "Update Merge Request",
    componentName: "gitlab.updateMergeRequest",
    isCollapsed: false,
    configuration: {
      project: "123",
      mergeRequestIid: "42",
      title: "feat: add login page (updated)",
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
