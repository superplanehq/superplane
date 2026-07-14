import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
} from "../types";
import { createIssueCommentMapper } from "./create_issue_comment";

const definition: ComponentDefinition = {
  name: "github.createIssueComment",
  label: "Create Issue Comment",
  description: "",
  icon: "github",
  color: "gray",
};

describe("createIssueCommentMapper.getExecutionDetails", () => {
  it("shows created comment details", () => {
    const details = createIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { repository: "superplane", issueNumber: "42", body: "Hi" },
        outputs: {
          default: [
            {
              data: {
                id: 1,
                html_url: "https://github.com/superplanehq/superplane/issues/42#issuecomment-1",
                created_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      URL: "https://github.com/superplanehq/superplane/issues/42#issuecomment-1",
    });
  });

  it("handles missing outputs", () => {
    const details = createIssueCommentMapper.getExecutionDetails(
      buildDetailsContext({ configuration: { repository: "superplane" }, outputs: {} }),
    );

    expect(details).toEqual({});
  });
});

describe("createIssueCommentMapper.props", () => {
  it("shows the repository from configuration", () => {
    const props = createIssueCommentMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { repository: "superplane" } }) }),
    );

    expect(props.metadata).toEqual([{ icon: "book", label: "superplane" }]);
  });

  it("coerces an IntegrationResourceRef repository object to its display name instead of crashing", () => {
    const props = createIssueCommentMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { repository: { id: "repo-id", name: "superplane", type: "repository" } },
        }),
      }),
    );

    expect(props.metadata).toEqual([{ icon: "book", label: "superplane" }]);
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Issue Comment",
    componentName: "github.createIssueComment",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create Issue Comment",
    componentName: "github.createIssueComment",
    isCollapsed: false,
    metadata: {
      repository: {
        id: "123456",
        name: "superplane",
        url: "https://github.com/superplanehq/superplane",
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
