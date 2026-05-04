import { describe, expect, it } from "vitest";
import { addCustomDomainMapper } from "./add_custom_domain";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Add Custom Domain",
  componentName: "render.service.addCustomDomain",
  isCollapsed: false,
};

const DEFINITION = {
  name: "render.service.addCustomDomain",
  label: "Add Custom Domain",
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
  const execution: ExecutionInfo = {
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
    outputs: outputData ? { default: [{ data: outputData, type: "render.customDomain.added" }] } : undefined,
  };
  return { nodes: [], node: NODE, execution };
}

describe("addCustomDomainMapper.props", () => {
  it("does not throw when node.configuration is undefined", () => {
    const ctx = makePropsContext({ configuration: undefined });
    expect(() => addCustomDomainMapper.props!(ctx)).not.toThrow();
    const props = addCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual([]);
  });

  it("does not throw when node.configuration is null", () => {
    const ctx = makePropsContext({ configuration: null as unknown as undefined });
    expect(() => addCustomDomainMapper.props!(ctx)).not.toThrow();
  });

  it("includes service metadata when configuration.service is set", () => {
    const ctx = makePropsContext({ configuration: { service: "srv-123" } });
    const props = addCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Service: srv-123" })]));
  });

  it("prefers service name from node metadata", () => {
    const ctx = makePropsContext({
      configuration: { service: "srv-123" },
      metadata: { service: { id: "srv-123", name: "backend-api" } },
    });
    const props = addCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ label: "Service: backend-api" })]),
    );
  });

  it("includes domain metadata when configuration.domain is set", () => {
    const ctx = makePropsContext({ configuration: { domain: "app.example.com" } });
    const props = addCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "app.example.com" })]));
  });

  it("includes wait for verification metadata when waitForVerification is true", () => {
    const ctx = makePropsContext({ configuration: { waitForVerification: true } });
    const props = addCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ label: "Wait for verification" })]),
    );
  });
});

describe("addCustomDomainMapper.getExecutionDetails", () => {
  it("returns dash values when there are no outputs", () => {
    const ctx = makeDetailsContext();
    const details = addCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain ID"]).toBe("-");
    expect(details["Domain Name"]).toBe("-");
    expect(details["Service ID"]).toBe("-");
    expect(details["Verification Status"]).toBe("-");
  });

  it("returns domain details from default output", () => {
    const ctx = makeDetailsContext({
      id: "cdm-abc123",
      name: "app.example.com",
      serviceId: "srv-xyz789",
      verificationStatus: "verified",
    });
    const details = addCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain ID"]).toBe("cdm-abc123");
    expect(details["Domain Name"]).toBe("app.example.com");
    expect(details["Service ID"]).toBe("srv-xyz789");
    expect(details["Verification Status"]).toBe("verified");
  });
});
