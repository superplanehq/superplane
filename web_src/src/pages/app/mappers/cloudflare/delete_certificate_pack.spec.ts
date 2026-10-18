import { describe, expect, it } from "bun:test";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
} from "../types";
import { deleteCertificatePackMapper } from "./delete_certificate_pack";
import { eventStateRegistry } from "./index";
import { buildExecution, createNodeFixtures } from "./test_support";

const { buildNode } = createNodeFixtures("cloudflare.deleteCertificatePack");

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode({ ...overrides?.node, componentName: "cloudflare.deleteCertificatePack" });
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const deleteDefinition: ComponentDefinition = {
  name: "cloudflare.deleteCertificatePack",
  label: "Delete Certificate Pack",
  description: "",
  icon: "shield-off",
  color: "orange",
};

function buildPropsCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode({ ...nodeOverrides, componentName: deleteDefinition.name }),
    componentDefinition: deleteDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("deleteCertificatePackMapper.props metadata", () => {
  it("shows the certificate pack picker display name when present", () => {
    const props = deleteCertificatePackMapper.props(
      buildPropsCtx({
        configuration: {
          certificatePack: "zone123/pack-abc",
          certificatePackDisplayName: "example.com - www.example.com",
        },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "shield-off", label: "example.com - www.example.com" }]);
  });

  it("shows the pack ID extracted from zone/pack value when display name is absent", () => {
    const props = deleteCertificatePackMapper.props(
      buildPropsCtx({
        configuration: { certificatePack: "zone123/pack-abc" },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "shield-off", label: "pack-abc" }]);
  });

  it("shows the stored readable resource name when there is no slash", () => {
    const props = deleteCertificatePackMapper.props(
      buildPropsCtx({
        configuration: { certificatePack: "example.com - www.example.com" },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "shield-off", label: "example.com - www.example.com" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = deleteCertificatePackMapper.props(buildPropsCtx({ configuration: {} }));
    expect(props.metadata).toEqual([]);
  });
});

describe("deleteCertificatePackMapper.getExecutionDetails", () => {
  it("shows zone, pack ID when hosts are absent, and deleted flag", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            {
              type: "cloudflare.certificate_pack.deleted",
              timestamp: new Date().toISOString(),
              data: { zoneId: "zone123", zoneName: "zone.example.com", packId: "pack-abc", deleted: true },
            },
          ],
        },
      },
    });
    const details = deleteCertificatePackMapper.getExecutionDetails(ctx);
    expect(details["Zone"]).toBe("zone.example.com");
    expect(details["Pack ID"]).toBe("pack-abc");
    expect(details["Hosts"]).toBeUndefined();
    expect(details["Deleted"]).toBe("Yes");
  });

  it("shows hosts instead of pack ID when hosts are present", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            {
              type: "cloudflare.certificate_pack.deleted",
              timestamp: new Date().toISOString(),
              data: {
                zoneId: "z1",
                zoneName: "example.com",
                packId: "pack-id",
                hosts: ["a.example.com", "b.example.com"],
                deleted: true,
              },
            },
          ],
        },
      },
    });
    const details = deleteCertificatePackMapper.getExecutionDetails(ctx);
    expect(details["Zone"]).toBe("example.com");
    expect(details["Hosts"]).toBe("a.example.com, b.example.com");
    expect(details["Pack ID"]).toBeUndefined();
    expect(details["Deleted"]).toBe("Yes");
  });

  it("falls back to zone id when zone name is missing", () => {
    const ctx = buildDetailsCtx({
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
    expect(details["Zone"]).toBe("zone123");
    expect(details["Pack ID"]).toBe("pack-abc");
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: undefined },
    });
    expect(() => deleteCertificatePackMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("eventStateRegistry.deleteCertificatePack", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteCertificatePack.getState(buildExecution())).toBe("deleted");
  });
});
