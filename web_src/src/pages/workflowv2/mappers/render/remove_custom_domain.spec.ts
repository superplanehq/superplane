import { describe, expect, it } from "vitest";
import { removeCustomDomainMapper } from "./remove_custom_domain";
import type { ComponentBaseContext, ExecutionDetailsContext, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Remove Custom Domain",
  componentName: "render.removeCustomDomain",
  isCollapsed: false,
};

const DEFINITION = {
  name: "render.removeCustomDomain",
  label: "Remove Custom Domain",
  description: "",
  icon: "globe",
  color: "gray",
};

function makePropsContext(overrides: Partial<NodeInfo> = {}): ComponentBaseContext {
  return {
    nodes: [],
    node: { ...NODE, ...overrides },
    componentDefinition: DEFINITION,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

function makeDetailsContext(outputData?: Record<string, unknown>): ExecutionDetailsContext {
  const execution: any = {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_SUCCEEDED",
    resultReason: "RESULT_REASON_UNSPECIFIED",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs: outputData ? { default: [{ data: outputData, type: "render.customDomain.removed" }] } : undefined,
  };
  return { nodes: [], node: NODE, execution };
}

describe("removeCustomDomainMapper.props", () => {
  it("does not throw when node.configuration is undefined", () => {
    const ctx = makePropsContext({ configuration: undefined });
    expect(() => removeCustomDomainMapper.props!(ctx)).not.toThrow();
    const props = removeCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual([]);
  });

  it("does not throw when node.configuration is null", () => {
    const ctx = makePropsContext({ configuration: null as unknown as undefined });
    expect(() => removeCustomDomainMapper.props!(ctx)).not.toThrow();
  });

  it("includes service metadata when configuration.service is set", () => {
    const ctx = makePropsContext({ configuration: { service: "srv-123" } });
    const props = removeCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Service: srv-123" })]));
  });

  it("includes domainName metadata when configuration.domainName is set", () => {
    const ctx = makePropsContext({ configuration: { domainName: "app.example.com" } });
    const props = removeCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "app.example.com" })]));
  });
});

describe("removeCustomDomainMapper.getExecutionDetails", () => {
  it("returns dash values when there are no outputs", () => {
    const ctx = makeDetailsContext();
    const details = removeCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain Name"]).toBe("-");
    expect(details["Service ID"]).toBe("-");
  });

  it("returns domain details from default output", () => {
    const ctx = makeDetailsContext({
      name: "app.example.com",
      serviceId: "srv-xyz789",
    });
    const details = removeCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain Name"]).toBe("app.example.com");
    expect(details["Service ID"]).toBe("srv-xyz789");
  });
});
