import { describe, expect, it } from "vitest";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { waitForPullRequestChecksMapper } from "./wait_for_pull_request_checks";

const REVISION = "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44";

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

function makeExecution(overrides: Partial<ExecutionInfo> = {}): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    rootEvent: undefined,
    metadata: {},
    configuration: {},
    ...overrides,
  };
}

function specBadge(specs: { title?: string; values?: Array<{ badges?: Array<{ label?: string }> }> }[], index: number) {
  return {
    title: specs[index]?.title,
    label: specs[index]?.values?.[0]?.badges?.[0]?.label,
  };
}

describe("github wait_for_pull_request_checks mapper", () => {
  it("shows the repository, revision, and selected check names", () => {
    const props = waitForPullRequestChecksMapper.props(
      makeContext({
        ref: REVISION,
        checkNames: ["DCO", "build"],
      }),
    );

    expect(props.metadata?.map((item) => item.label)).toEqual(["hello", "d6f3c8a", "DCO, build"]);
  });

  it("shows pending and failed checks from execution metadata", () => {
    const props = waitForPullRequestChecksMapper.props(
      makeContext(
        { ref: REVISION },
        makeExecution({
          metadata: {
            sha: REVISION,
            selectedChecks: [
              { name: "DCO", status: "completed", conclusion: "success" },
              { name: "build", status: "in_progress" },
            ],
            failedChecks: [{ name: "lint", status: "completed", conclusion: "failure" }],
          },
        }),
      ),
    );
    const specs = props.specs ?? [];

    expect(specBadge(specs, 0)).toEqual({ title: "pending", label: "build" });
    expect(specBadge(specs, 1)).toEqual({ title: "failed", label: "lint" });
  });

  it("summarizes repository, revision, pending checks, and failed checks", () => {
    const node = makeNode({ ref: REVISION });
    const context: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: makeExecution({
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        outputs: {
          failed: [
            {
              data: {
                repository: "hello",
                sha: REVISION,
                selectedChecks: [
                  { name: "DCO", status: "completed", conclusion: "success" },
                  { name: "build", status: "completed", conclusion: "failure" },
                ],
                failedChecks: [{ name: "build", status: "completed", conclusion: "failure" }],
              },
            },
          ],
        },
      }),
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
