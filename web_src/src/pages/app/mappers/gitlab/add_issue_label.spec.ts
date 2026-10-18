import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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
  return buildGitLabDetailsContext(
    {
      name: "Add Issue Label",
      componentName: "gitlab.addIssueLabel",
      configuration: {
        project: "123",
        issueIid: "1",
      },
    },
    execution,
  );
}
