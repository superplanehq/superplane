import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getPipelineMinutesUsageMapper } from "./get_pipeline_minutes_usage";

describe("getPipelineMinutesUsageMapper", () => {
  it("shows pipeline minutes usage details", () => {
    const details = getPipelineMinutesUsageMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {
          default: [
            {
              data: {
                month: "July",
                monthIso8601: "2026-07-01",
                minutes: 245,
                sharedRunnersDuration: 14700,
                projects: [
                  { minutes: 180, sharedRunnersDuration: 10800, project: { name: "hello-world" } },
                  { minutes: 65, sharedRunnersDuration: 3900, project: { name: "other-project" } },
                ],
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Month: "July",
      "Minutes Used": "245",
      "Shared Runners Duration": "14700s",
      Projects: "hello-world, other-project",
    });
  });

  it("handles missing outputs", () => {
    const details = getPipelineMinutesUsageMapper.getExecutionDetails(
      buildDetailsContext({
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows the selected project count in node metadata", () => {
    const context = buildDetailsContext({});
    context.node.configuration = { projects: ["1", "2"] };

    const props = getPipelineMinutesUsageMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.getPipelineMinutesUsage",
        label: "Get Pipeline Minutes Usage",
        description: "",
        icon: "gitlab",
        color: "orange",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    });

    expect(props.metadata).toEqual([{ icon: "book", label: "2 project(s)" }]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Pipeline Minutes Usage",
    componentName: "gitlab.getPipelineMinutesUsage",
    isCollapsed: false,
    configuration: {},
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
