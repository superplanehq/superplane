import { describe, expect, it } from "vitest";
import { triggerDNSConfigurationMapper } from "./trigger_dns_configuration";
import type { ComponentBaseContext, ExecutionDetailsContext, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Trigger DNS Configuration",
  componentName: "render.triggerDNSConfiguration",
  isCollapsed: false,
};

const DEFINITION = {
  name: "render.triggerDNSConfiguration",
  label: "Trigger DNS Configuration",
  description: "",
  icon: "shield-check",
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
  const execution: ExecutionDetailsContext["execution"] = {
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
    outputs: outputData
      ? { default: [{ data: outputData, type: "render.dnsConfiguration.verification.requested" }] }
      : undefined,
  };
  return { nodes: [], node: NODE, execution };
}

describe("triggerDNSConfigurationMapper.props", () => {
  it("does not throw when node.configuration is undefined", () => {
    const ctx = makePropsContext({ configuration: undefined });
    expect(() => triggerDNSConfigurationMapper.props!(ctx)).not.toThrow();
    const props = triggerDNSConfigurationMapper.props!(ctx);
    expect(props.metadata).toEqual([]);
  });

  it("does not throw when node.configuration is null", () => {
    const ctx = makePropsContext({ configuration: null as unknown as undefined });
    expect(() => triggerDNSConfigurationMapper.props!(ctx)).not.toThrow();
  });

  it("includes service metadata when configuration.service is set", () => {
    const ctx = makePropsContext({ configuration: { service: "srv-123" } });
    const props = triggerDNSConfigurationMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Service: srv-123" })]));
  });

  it("prefers service name from node metadata", () => {
    const ctx = makePropsContext({
      configuration: { service: "srv-123" },
      metadata: { service: { id: "srv-123", name: "backend-api" } },
    });
    const props = triggerDNSConfigurationMapper.props!(ctx);
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ label: "Service: backend-api" })]),
    );
  });

  it("includes domainName metadata when configuration.domainName is set", () => {
    const ctx = makePropsContext({ configuration: { domainName: "app.example.com" } });
    const props = triggerDNSConfigurationMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "app.example.com" })]));
  });
});

describe("triggerDNSConfigurationMapper.getExecutionDetails", () => {
  it("returns dash values when there are no outputs", () => {
    const ctx = makeDetailsContext();
    const details = triggerDNSConfigurationMapper.getExecutionDetails!(ctx);
    expect(details["Domain Name"]).toBe("-");
    expect(details["Service ID"]).toBe("-");
    expect(details.Status).toBe("-");
  });

  it("returns verification details from default output", () => {
    const ctx = makeDetailsContext({
      name: "app.example.com",
      serviceId: "srv-xyz789",
      status: "accepted",
    });
    const details = triggerDNSConfigurationMapper.getExecutionDetails!(ctx);
    expect(details["Domain Name"]).toBe("app.example.com");
    expect(details["Service ID"]).toBe("srv-xyz789");
    expect(details.Status).toBe("accepted");
  });
});
