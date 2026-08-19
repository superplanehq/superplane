import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updatePullRequestMapper } from "./update_pull_request";

describe("updatePullRequestMapper", () => {
  it("shows the updated pull request details", () => {
    const details = updatePullRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          repository: "hello",
          pullNumber: "42",
        },
        outputs: {
          default: [
            {
              data: {
                number: 42,
                title: "Updated title",
                state: "closed",
                html_url: "https://github.com/testhq/hello/pull/42",
                base: { ref: "release" },
                labels: [{ name: "bug" }],
                assignees: [{ login: "octocat" }],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Repository: "hello",
      "Pull Request": "#42",
      "Pull Request URL": "https://github.com/testhq/hello/pull/42",
      Title: "Updated title",
      State: "closed",
      "Base Branch": "release",
      Labels: "bug",
      Assignees: "octocat",
    });
  });

  it("falls back to a dash when there is no output yet", () => {
    const details = updatePullRequestMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { repository: "hello", pullNumber: "42" },
        outputs: {},
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Repository: "hello",
      "Pull Request": "#42",
      "Pull Request URL": "https://github.com/testhq/hello/pull/42",
      Title: "-",
      State: "-",
      "Base Branch": "-",
      Labels: "-",
      Assignees: "-",
    });
  });

  it("shows repository and pull request number in node metadata", () => {
    const context = buildDetailsContext({});
    const props = updatePullRequestMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "github.updatePullRequest",
        label: "Update Pull Request",
        description: "",
        icon: "github",
        color: "gray",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    });

    expect(props.metadata).toEqual([
      { icon: "book", label: "hello" },
      { icon: "git-pull-request", label: "#42" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Update pull request",
    componentName: "github.updatePullRequest",
    isCollapsed: false,
    configuration: {
      repository: "hello",
      pullNumber: "42",
    },
    metadata: {
      repository: {
        id: "123456",
        name: "hello",
        url: "https://github.com/testhq/hello",
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
