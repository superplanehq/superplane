import { describe, expect, it } from "vitest";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { orderCertificatePackMapper } from "./order_certificate_pack";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.orderCertificatePack",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.certificate_pack.ordered",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
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
    ...overrides,
  };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode({ ...overrides?.node, componentName: "cloudflare.orderCertificatePack" });
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const orderDefinition: ComponentDefinition = {
  name: "cloudflare.orderCertificatePack",
  label: "Order Certificate Pack",
  description: "",
  icon: "shield-check",
  color: "orange",
};

function buildPropsCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode({ ...nodeOverrides, componentName: orderDefinition.name }),
    componentDefinition: orderDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("orderCertificatePackMapper.props metadata", () => {
  it("shows single host directly when only one host configured", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          zone: "zone123",
          hosts: ["preview.example.com"],
          certificateAuthority: "lets_encrypt",
          validationMethod: "txt",
        },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "shield-check", label: "preview.example.com" });
  });

  it("shows host count when multiple hosts configured", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          zone: "zone123",
          hosts: ["example.com", "*.example.com"],
          certificateAuthority: "lets_encrypt",
          validationMethod: "txt",
        },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "shield-check", label: "2 hosts" });
  });

  it("shows certificate authority label", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          hosts: ["example.com"],
          certificateAuthority: "custom_ca_provider",
        },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "award", label: "custom ca provider" });
  });

  it("shows validity period for certificate authorities that support it", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          hosts: ["example.com"],
          certificateAuthority: "google",
          validityDays: "90",
        },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "calendar-clock", label: "3 months" });
  });

  it("does not show validity period for certificate authorities that do not support it", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          hosts: ["example.com"],
          certificateAuthority: "lets_encrypt",
          validityDays: "90",
        },
      }),
    );
    expect(props.metadata).not.toContainEqual({ icon: "calendar-clock", label: "3 months" });
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = orderCertificatePackMapper.props(buildPropsCtx({ configuration: {} }));
    expect(props.metadata).toEqual([]);
  });
});

describe("orderCertificatePackMapper.getExecutionDetails", () => {
  it("shows zone name, hosts, and omits pack ID when hosts are present", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              zoneName: "example.com",
              packId: "pack-abc",
              pack: {
                id: "pack-abc",
                certificate_authority: "lets_encrypt",
                hosts: ["preview.example.com"],
                status: "initializing",
                type: "advanced",
                validation_method: "txt",
                validity_days: 90,
              },
            }),
          ],
        },
      },
    });
    const details = orderCertificatePackMapper.getExecutionDetails(ctx);
    expect(details["Zone"]).toBe("example.com");
    expect(details["Pack ID"]).toBeUndefined();
    expect(details["Status"]).toBe("initializing");
    expect(details["CA"]).toBe("lets_encrypt");
    expect(details["Validation"]).toBe("txt");
    expect(details["Validity"]).toBe("3 months");
    expect(details["Hosts"]).toBe("preview.example.com");
  });

  it("falls back to zone id when zone name is absent", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              packId: "pack-only",
              pack: {
                id: "pack-only",
                certificate_authority: "lets_encrypt",
                hosts: [],
                status: "initializing",
                type: "advanced",
                validation_method: "txt",
              },
            }),
          ],
        },
      },
    });
    const details = orderCertificatePackMapper.getExecutionDetails(ctx);
    expect(details["Zone"]).toBe("zone123");
    expect(details["Pack ID"]).toBe("pack-only");
    expect(details["Hosts"]).toBeUndefined();
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: undefined },
    });
    expect(() => orderCertificatePackMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("eventStateRegistry.orderCertificatePack", () => {
  it("maps finished success to ordered", () => {
    expect(eventStateRegistry.orderCertificatePack.getState(buildExecution())).toBe("ordered");
  });

  it("returns running when in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.orderCertificatePack.getState(running)).toBe("running");
  });
});
