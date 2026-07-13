import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getIssueMapper } from "./get_issue";

describe("getIssueMapper", () => {
  it("shows retrieved issue details", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "7",
        },
        outputs: {
          default: [
            {
              type: "gitlab.issue",
              timestamp: "2026-02-12T09:12:03.000Z",
              data: {
                id: 41,
                iid: 7,
                project_id: 123,
                title: "Login page rendering issue",
                state: "opened",
                labels: ["bug", "frontend"],
                author: { id: 22, name: "Jamie Rivera", username: "jrivera" },
                web_url: "https://gitlab.com/my-group/my-project/-/issues/7",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Retrieved At": expect.any(String),
      Issue: "#7 Login page rendering issue",
      "Issue URL": "https://gitlab.com/my-group/my-project/-/issues/7",
      State: "opened",
      Author: "jrivera",
      Labels: "bug, frontend",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.issue",
              timestamp: "2026-02-12T09:12:03.000Z",
              data: {
                id: 41,
                iid: 7,
                title: "Login page rendering issue",
                state: "opened",
                labels: ["bug", "frontend"],
                author: { username: "jrivera" },
                web_url: "https://gitlab.com/my-group/my-project/-/issues/7",
              },
            },
          ],
        },
      }),
    );

    const keys = Object.keys(details);
    expect(keys[0]).toBe("Retrieved At");
    expect(keys.length).toBeLessThanOrEqual(6);
  });

  it("omits labels when the issue has none", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.issue",
              timestamp: "2026-02-12T09:12:03.000Z",
              data: { id: 41, iid: 7, title: "Login page rendering issue", state: "opened", labels: [] },
            },
          ],
        },
      }),
    );

    expect(details["Labels"]).toBeUndefined();
    expect(details["Issue"]).toBe("#7 Login page rendering issue");
  });

  it("handles missing outputs", () => {
    const details = getIssueMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "7" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows issue subtitle from the output", () => {
    const subtitle = getIssueMapper.subtitle(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.issue",
              timestamp: "2026-02-12T09:12:03.000Z",
              data: { id: 41, iid: 7, title: "Login page rendering issue", state: "opened" },
            },
          ],
        },
      }),
    );

    expect(subtitle).toBe("#7 Login page rendering issue");
  });

  it("shows project and issue IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = getIssueMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.getIssue",
        label: "Get Issue",
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
      { icon: "circle-dot", label: "#7" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Issue",
    componentName: "gitlab.getIssue",
    isCollapsed: false,
    configuration: {
      project: "123",
      issueIid: "7",
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
