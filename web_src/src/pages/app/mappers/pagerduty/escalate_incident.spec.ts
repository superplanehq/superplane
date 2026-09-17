import { describe, expect, it } from "bun:test";
import { escalateIncidentMapper } from "./escalate_incident";
import type { ComponentBaseContext, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Escalate Incident",
  componentName: "pagerduty.escalateIncident",
  isCollapsed: false,
};

const DEFINITION = {
  name: "pagerduty.escalateIncident",
  label: "Escalate Incident",
  description: "",
  icon: "zap",
  color: "blue",
};

function makeContext(overrides: Partial<NodeInfo> = {}): ComponentBaseContext {
  return {
    nodes: [],
    node: { ...NODE, ...overrides },
    componentDefinition: DEFINITION,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("pagerduty escalateIncidentMapper.props", () => {
  it("shows Next level when escalationLevel is missing", () => {
    const props = escalateIncidentMapper.props!(makeContext({ configuration: { incidentId: "P123" } }));
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Next level" })]));
  });

  it("shows the configured string escalation level", () => {
    const props = escalateIncidentMapper.props!(
      makeContext({ configuration: { incidentId: "P123", escalationLevel: "2" } }),
    );
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Level: 2" })]));
  });
});
