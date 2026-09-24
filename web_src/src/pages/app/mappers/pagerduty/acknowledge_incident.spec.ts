import { describe, expect, it } from "bun:test";
import { acknowledgeIncidentMapper } from "./acknowledge_incident";
import { defaultTriggerRenderer } from "../default";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Acknowledge Incident",
  componentName: "pagerduty.acknowledgeIncident",
  isCollapsed: false,
  configuration: {},
};

const DEFINITION = {
  name: "pagerduty.acknowledgeIncident",
  label: "Acknowledge Incident",
  description: "",
  icon: "zap",
  color: "blue",
};

function makeExecution(): ExecutionInfo {
  const createdAt = new Date().toISOString();
  return {
    id: "exec-1",
    createdAt,
    updatedAt: createdAt,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt,
      data: {},
      nodeId: "missing-trigger",
      type: "pagerduty.onIncident",
    },
  };
}

function makeContext(nodes: NodeInfo[]): ComponentBaseContext {
  return {
    nodes,
    node: NODE,
    componentDefinition: DEFINITION,
    lastExecutions: [makeExecution()],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("pagerduty acknowledgeIncidentMapper.props", () => {
  it("uses the default trigger renderer when the root trigger node is absent", () => {
    const execution = makeExecution();
    const props = acknowledgeIncidentMapper.props(makeContext([]));
    const expectedTitle = defaultTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent }).title;

    expect(props.eventSections).toHaveLength(1);
    expect(props.eventSections![0].eventTitle).toBe(expectedTitle);
  });
});
