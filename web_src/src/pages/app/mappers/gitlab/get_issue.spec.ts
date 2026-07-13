import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getIssueMapper } from "./get_issue";

describe("getIssueMapper", () => {
  it("shows fetched issue details", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "1",
        },
        outputs: {
          default: [
            {
              data: {
                id: 101,
                iid: 1,
                title: "Bug report",
                state: "opened",
                web_url: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
                author: { username: "root" },
                created_at: "2026-06-11T14:30:00Z",
                labels: ["bug"],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      IID: "1",
      ID: "101",
      State: "opened",
      URL: "https://gitlab.com/felixgateru/hello-world/-/issues/1",
      Title: "Bug report",
      "Created At": expect.any(String),
      "Created By": "root",
      Labels: "bug",
    });
  });

  it("handles missing outputs", () => {
    const details = getIssueMapper.getExecutionDetails(
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
    name: "Get Issue",
    componentName: "gitlab.getIssue",
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
