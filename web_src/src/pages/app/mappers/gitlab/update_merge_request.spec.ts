import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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

    expect(Object.keys(details)[0]).toBe("Updated At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(details).toEqual({
      "Updated At": expect.any(String),
      "Merge Request": "!42 feat: add login page (updated)",
      "Merge Request URL": "https://gitlab.com/my-group/my-project/-/merge_requests/42",
      "Source Branch": "feature/login-page",
      "Target Branch": "develop",
      State: "opened",
    });
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

    expect(details["Updated At"]).toBe(new Date("2026-02-14T09:12:33.000Z").toLocaleString());
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
  return buildGitLabDetailsContext(
    {
      name: "Update Merge Request",
      componentName: "gitlab.updateMergeRequest",
      configuration: {
        project: "123",
        mergeRequestIid: "42",
        title: "feat: add login page (updated)",
      },
    },
    execution,
  );
}
