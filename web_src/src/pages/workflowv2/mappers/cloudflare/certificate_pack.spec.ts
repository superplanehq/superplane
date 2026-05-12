import { describe, expect, it } from "vitest";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { orderCertificatePackMapper, deleteCertificatePackMapper } from "./certificate_pack";
import { eventStateRegistry } from "./index";

// ── Helpers ───────────────────────────────────────────────────────────────────

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

function buildDetailsCtx(
  componentName: string,
  overrides?: {
    node?: Partial<NodeInfo>;
    execution?: Partial<ExecutionInfo>;
  },
): ExecutionDetailsContext {
  const node = buildNode({ ...overrides?.node, componentName });
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const orderDefinition: ComponentDefinition = {
  name: "cloudflare.orderCertificatePack",
  label: "Order Certificate Pack",
  description: "",
  icon: "shield-check",
  color: "orange",
};

const deleteDefinition: ComponentDefinition = {
  name: "cloudflare.deleteCertificatePack",
  label: "Delete Certificate Pack",
  description: "",
  icon: "shield-off",
  color: "orange",
};

function buildPropsCtx(definition: ComponentDefinition, nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode({ ...nodeOverrides, componentName: definition.name }),
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

// ── orderCertificatePackMapper.props ──────────────────────────────────────────

describe("orderCertificatePackMapper.props metadata", () => {
  it("shows single host directly when only one host configured", () => {
    const props = orderCertificatePackMapper.props(
      buildPropsCtx(orderDefinition, {
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
      buildPropsCtx(orderDefinition, {
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
      buildPropsCtx(orderDefinition, {
        configuration: {
          hosts: ["example.com"],
          certificateAuthority: "lets_encrypt",
        },
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "award", label: "lets encrypt" });
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = orderCertificatePackMapper.props(buildPropsCtx(orderDefinition, { configuration: {} }));
    expect(props.metadata).toEqual([]);
  });
});

// ── orderCertificatePackMapper.getExecutionDetails ────────────────────────────

describe("orderCertificatePackMapper.getExecutionDetails", () => {
  it("extracts pack fields from output", () => {
    const ctx = buildDetailsCtx("cloudflare.orderCertificatePack", {
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              packId: "pack-abc",
              pack: {
                id: "pack-abc",
                certificate_authority: "lets_encrypt",
                hosts: ["preview.example.com"],
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
    expect(details["Pack ID"]).toBe("pack-abc");
    expect(details["Status"]).toBe("initializing");
    expect(details["CA"]).toBe("lets_encrypt");
    expect(details["Validation"]).toBe("txt");
    expect(details["Hosts"]).toBe("preview.example.com");
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx("cloudflare.orderCertificatePack", {
      execution: { outputs: undefined },
    });
    expect(() => orderCertificatePackMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

// ── deleteCertificatePackMapper.props ─────────────────────────────────────────

describe("deleteCertificatePackMapper.props metadata", () => {
  it("shows the pack ID extracted from zone/pack value", () => {
    const props = deleteCertificatePackMapper.props(
      buildPropsCtx(deleteDefinition, {
        configuration: { certificatePack: "zone123/pack-abc" },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "shield-off", label: "pack-abc" }]);
  });

  it("shows the raw value when there is no slash", () => {
    const props = deleteCertificatePackMapper.props(
      buildPropsCtx(deleteDefinition, {
        configuration: { certificatePack: "pack-only-id" },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "shield-off", label: "pack-only-id" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = deleteCertificatePackMapper.props(buildPropsCtx(deleteDefinition, { configuration: {} }));
    expect(props.metadata).toEqual([]);
  });
});

// ── deleteCertificatePackMapper.getExecutionDetails ───────────────────────────

describe("deleteCertificatePackMapper.getExecutionDetails", () => {
  it("shows pack ID, zone ID and deleted flag", () => {
    const ctx = buildDetailsCtx("cloudflare.deleteCertificatePack", {
      execution: {
        outputs: {
          default: [
            {
              type: "cloudflare.certificate_pack.deleted",
              timestamp: new Date().toISOString(),
              data: { zoneId: "zone123", packId: "pack-abc", deleted: true },
            },
          ],
        },
      },
    });
    const details = deleteCertificatePackMapper.getExecutionDetails(ctx);
    expect(details["Pack ID"]).toBe("pack-abc");
    expect(details["Zone ID"]).toBe("zone123");
    expect(details["Deleted"]).toBe("Yes");
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx("cloudflare.deleteCertificatePack", {
      execution: { outputs: undefined },
    });
    expect(() => deleteCertificatePackMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

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

describe("eventStateRegistry.deleteCertificatePack", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteCertificatePack.getState(buildExecution())).toBe("deleted");
  });

  it("returns failed on component failure", () => {
    const failed = buildExecution({
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.deleteCertificatePack.getState(failed)).toBe("failed");
  });
});
