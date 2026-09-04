import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { FIND_PULL_REQUEST_STATE_REGISTRY, findPullRequestMapper } from "./find_pull_request";

describe("findPullRequestMapper", () => {
  it("shows the found pull request details", () => {
    const details = findPullRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          found: [
            {
              data: {
                number: 42,
                title: "Add new feature",
                state: "open",
                html_url: "https://github.com/testhq/hello/pull/42",
                head: { ref: "feature", sha: "abc" },
                base: { ref: "main" },
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Result: "Found",
      "Pull Request": "#42",
      Title: "Add new feature",
      State: "open",
      Branches: "feature → main",
      "Pull Request URL": "https://github.com/testhq/hello/pull/42",
    });
  });

  it("shows a not found result when no pull request matches", () => {
    const details = findPullRequestMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          notFound: [{ data: { repository: "hello", head: "feature" } }],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Result: "Not Found",
    });
  });
});

describe("FIND_PULL_REQUEST_STATE_REGISTRY", () => {
  it("resolves to found when the found channel has output", () => {
    const state = FIND_PULL_REQUEST_STATE_REGISTRY.getState(buildExecution({ found: [{ data: {} }] }));
    expect(state).toBe("found");
  });

  it("resolves to notFound when the notFound channel has output", () => {
    const state = FIND_PULL_REQUEST_STATE_REGISTRY.getState(buildExecution({ notFound: [{ data: {} }] }));
    expect(state).toBe("notFound");
  });

  it("resolves to neutral when there is no output yet", () => {
    const state = FIND_PULL_REQUEST_STATE_REGISTRY.getState(buildExecution({}));
    expect(state).toBe("neutral");
  });
});

function buildExecution(outputs: Record<string, unknown>): ExecutionInfo {
  return {
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
    outputs,
  };
}

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Find pull request",
    componentName: "github.findPullRequest",
    isCollapsed: false,
    configuration: {
      repository: "hello",
      head: "feature",
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
    execution: { ...buildExecution({}), ...execution },
  };
}
