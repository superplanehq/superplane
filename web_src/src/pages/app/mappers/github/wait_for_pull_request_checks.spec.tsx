import { describe, expect, it } from "vitest";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { waitForPullRequestChecksMapper } from "./wait_for_pull_request_checks";

function makeNode(configuration: unknown, metadata: unknown = {}): NodeInfo {
  return {
    id: "node-1",
    name: "Wait For Pull Request Checks",
    componentName: "github.waitForPullRequestChecks",
    isCollapsed: false,
    configuration,
    metadata,
  };
}

function makeContext(configuration: unknown, execution?: ExecutionInfo): ComponentBaseContext {
  return {
    nodes: [],
    node: makeNode(configuration, { repository: { name: "hello" } }),
    componentDefinition: {
      name: "waitForPullRequestChecks",
      label: "Wait For Pull Request Checks",
      description: "",
      icon: "github",
      color: "gray",
    },
    lastExecutions: execution ? [execution] : [],
    currentUser: {
      id: "123",
      name: "John Doe",
      email: "john.doe@example.com",
      roles: ["admin"],
      groups: ["developers"],
    },
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

describe("github wait_for_pull_request_checks mapper", () => {
  it("shows the repository, revision, and selected check names", () => {
    const props = waitForPullRequestChecksMapper.props(
      makeContext({
        ref: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
        checkNames: ["DCO", "build"],
      }),
    );

    expect(props.metadata?.map((item) => item.label)).toEqual(["hello", "d6f3c8a", "DCO, build"]);
  });

  it("shows pending and failed checks from execution metadata", () => {
    const props = waitForPullRequestChecksMapper.props(
      makeContext(
        { ref: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44" },
        {
          id: "exec-1",
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
          resultReason: "RESULT_REASON_OK",
          resultMessage: "",
          metadata: {
            sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
            selectedChecks: [
              { name: "DCO", status: "completed", conclusion: "success" },
              { name: "build", status: "in_progress" },
            ],
            failedChecks: [{ name: "lint", status: "completed", conclusion: "failure" }],
          },
          configuration: {},
        },
      ),
    );

    expect(props.specs?.[0]?.title).toBe("pending");
    expect(props.specs?.[0]?.values?.[0]?.badges?.[0]?.label).toBe("build");
    expect(props.specs?.[1]?.title).toBe("failed");
    expect(props.specs?.[1]?.values?.[0]?.badges?.[0]?.label).toBe("lint");
  });

  it("summarizes repository, revision, pending checks, and failed checks", () => {
    const node = makeNode({ ref: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44" });
    const context: ExecutionDetailsContext = {
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
        outputs: {
          failed: [
            {
              data: {
                repository: "hello",
                sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
                selectedChecks: [
                  { name: "DCO", status: "completed", conclusion: "success" },
                  { name: "build", status: "completed", conclusion: "failure" },
                ],
                failedChecks: [{ name: "build", status: "completed", conclusion: "failure" }],
              },
            },
          ],
        },
      },
    };

    expect(waitForPullRequestChecksMapper.getExecutionDetails(context)).toEqual({
      Repository: "hello",
      Revision: "d6f3c8a",
      "Selected checks": "2",
      "Pending checks": "None",
      "Failed checks": "build",
    });
  });
});
