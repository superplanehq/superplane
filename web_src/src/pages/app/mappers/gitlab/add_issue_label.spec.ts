import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { addIssueLabelMapper } from "./add_issue_label";

describe("addIssueLabelMapper", () => {
  it("shows the labels currently on the issue after the addition", () => {
    const details = addIssueLabelMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          issueIid: "1",
          labels: ["needs-triage"],
        },
        outputs: {
          default: [
            {
              type: "gitlab.labels",
              timestamp: "2026-02-13T11:16:17.520Z",
              data: ["bug", "urgent", "needs-triage"],
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": expect.any(String),
      Labels: "bug, urgent, needs-triage",
    });
  });

  it("handles missing outputs", () => {
    const details = addIssueLabelMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", issueIid: "1" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and issue IID in node metadata", () => {
    const context = buildDetailsContext({});
    const props = addIssueLabelMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.addIssueLabel",
        label: "Add Issue Label",
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
    name: "Add Issue Label",
    componentName: "gitlab.addIssueLabel",
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
