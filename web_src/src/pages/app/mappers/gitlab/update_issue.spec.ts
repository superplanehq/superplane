import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updateIssueMapper } from "./update_issue";

describe("updateIssueMapper", () => {
  it("shows updated issue details", () => {
    const details = updateIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "1",
          state: "close",
        },
        outputs: {
          default: [
            {
              data: {
                id: 101,
                iid: 1,
                title: "Bug report",
                state: "closed",
                web_url: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
                author: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
                closed_by: { username: "root" },
                closed_at: "2026-06-11T15:00:00Z",
                labels: ["bug"],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": expect.any(String),
      Title: "Bug report",
      State: "closed",
      "Created By": "root",
      Labels: "bug",
      URL: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
    });
  });

  it("handles missing outputs", () => {
    const details = updateIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Update Issue",
    componentName: "gitlab.updateIssue",
    isCollapsed: false,
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
