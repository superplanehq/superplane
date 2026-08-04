import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { removeIssueAssigneeMapper } from "./remove_issue_assignee";

describe("removeIssueAssigneeMapper", () => {
  it("shows updated issue assignee details after removal", () => {
    const details = removeIssueAssigneeMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.updateIssue",
              timestamp: "2023-01-01T10:05:00.000Z",
              data: {
                id: 1,
                iid: 1,
                project_id: 123,
                title: "Example Issue",
                state: "opened",
                updated_at: "2023-01-01T10:05:00.000Z",
                assignees: [
                  { id: 30, username: "amorgan", name: "Alex Morgan", state: "active", avatar_url: "", web_url: "" },
                ],
                web_url: "https://gitlab.com/my-group/my-project/-/issues/1",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": expect.any(String),
      Issue: "#1 Example Issue",
      "Issue URL": "https://gitlab.com/my-group/my-project/-/issues/1",
      Assignees: "@amorgan",
      State: "opened",
    });
  });

  it("shows the timestamp first and at most 6 details", () => {
    const details = removeIssueAssigneeMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.updateIssue",
              timestamp: "2023-01-01T10:05:00.000Z",
              data: {
                id: 1,
                iid: 1,
                title: "Example Issue",
                state: "opened",
                updated_at: "2023-01-01T10:05:00.000Z",
                assignees: [
                  { id: 30, username: "amorgan", name: "Alex Morgan", state: "active", avatar_url: "", web_url: "" },
                ],
                web_url: "https://gitlab.com/my-group/my-project/-/issues/1",
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

  it("shows None when there are no assignees left", () => {
    const details = removeIssueAssigneeMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              type: "gitlab.updateIssue",
              timestamp: "2023-01-01T10:05:00.000Z",
              data: { id: 1, iid: 1, title: "Example Issue", state: "opened", assignees: [] },
            },
          ],
        },
      }),
    );

    expect(details["Assignees"]).toBe("None");
  });

  it("handles missing outputs", () => {
    const details = removeIssueAssigneeMapper.getExecutionDetails(buildDetailsContext({ outputs: {} }));

    expect(details).toEqual({});
  });

  it("shows project and issue IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = removeIssueAssigneeMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.removeIssueAssignee",
        label: "Remove Issue Assignee",
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
      { icon: "circle-dot", label: "#1" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Remove Issue Assignee",
    componentName: "gitlab.removeIssueAssignee",
    isCollapsed: false,
    configuration: {
      project: "123",
      issueIid: "1",
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
