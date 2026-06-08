import { describe, expect, it } from "vitest";
import { createIncidentMapper } from "./create_incident";
import type { ComponentBaseContext, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Create Incident",
  componentName: "pagerduty.create_incident",
  isCollapsed: false,
};

const DEFINITION = {
  name: "pagerduty.create_incident",
  label: "Create Incident",
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

describe("pagerduty createIncidentMapper.props", () => {
  it("does not throw when node.configuration is null", () => {
    const ctx = makeContext({ configuration: null as unknown as undefined });
    expect(() => createIncidentMapper.props!(ctx)).not.toThrow();
    const props = createIncidentMapper.props!(ctx);
    expect(props.metadata).toEqual([]);
  });

  it("does not throw when node.configuration is undefined", () => {
    const ctx = makeContext({ configuration: undefined });
    expect(() => createIncidentMapper.props!(ctx)).not.toThrow();
  });

  it("includes urgency metadata when configuration.urgency is set", () => {
    const ctx = makeContext({ configuration: { urgency: "high" } });
    const props = createIncidentMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Urgency: high" })]));
  });
});
