import { describe, expect, it } from "vitest";
import { retrieveCustomDomainMapper } from "./retrieve_custom_domain";
import type { ComponentBaseContext, ExecutionDetailsContext, NodeInfo } from "../types";

const NODE: NodeInfo = {
  id: "n1",
  name: "Retrieve Custom Domain",
  componentName: "render.retrieveCustomDomain",
  isCollapsed: false,
};

const DEFINITION = {
  name: "render.retrieveCustomDomain",
  label: "Retrieve Custom Domain",
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
    outputs: outputData ? { default: [{ data: outputData, type: "render.customDomain" }] } : undefined,
  };
  return { nodes: [], node: NODE, execution };
}

describe("retrieveCustomDomainMapper.props", () => {
  it("does not throw when node.configuration is undefined", () => {
    const ctx = makePropsContext({ configuration: undefined });
    expect(() => retrieveCustomDomainMapper.props!(ctx)).not.toThrow();
    const props = retrieveCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual([]);
  });

  it("includes service and domain metadata", () => {
    const ctx = makePropsContext({
      configuration: { service: "srv-123", domainName: "app.example.com" },
      metadata: { service: { id: "srv-123", name: "backend-api" } },
    });
    const props = retrieveCustomDomainMapper.props!(ctx);
    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "Service: backend-api" }),
        expect.objectContaining({ label: "app.example.com" }),
      ]),
    );
  });
});

describe("retrieveCustomDomainMapper.getExecutionDetails", () => {
  it("returns dash values when there are no outputs", () => {
    const ctx = makeDetailsContext();
    const details = retrieveCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain ID"]).toBe("-");
    expect(details["Domain Name"]).toBe("-");
    expect(details["Service ID"]).toBe("-");
    expect(details["Verification Status"]).toBe("-");
  });

  it("returns custom domain details from default output", () => {
    const ctx = makeDetailsContext({
      id: "cdm-abc123",
      name: "app.example.com",
      serviceId: "srv-xyz789",
      verificationStatus: "verified",
    });
    const details = retrieveCustomDomainMapper.getExecutionDetails!(ctx);
    expect(details["Domain ID"]).toBe("cdm-abc123");
    expect(details["Domain Name"]).toBe("app.example.com");
    expect(details["Service ID"]).toBe("srv-xyz789");
    expect(details["Verification Status"]).toBe("verified");
  });
});
