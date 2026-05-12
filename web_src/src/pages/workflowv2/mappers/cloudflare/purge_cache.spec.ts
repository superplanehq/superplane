import { describe, expect, it } from "vitest";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { purgeCacheMapper } from "./purge_cache";
import { eventStateRegistry } from "./index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Purge CDN Cache",
    componentName: "cloudflare.purgeCache",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.cache.purged",
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
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.purgeCache",
  label: "Purge Cache",
  description: "",
  icon: "zap",
  color: "orange",
};

function buildPropsCtx(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

// ── props ─────────────────────────────────────────────────────────────────────

describe("purgeCacheMapper.props metadata", () => {
  it("shows purge everything label when mode is everything", () => {
    const props = purgeCacheMapper.props(buildPropsCtx({ node: buildNode({ configuration: { mode: "everything" } }) }));
    expect(props.metadata).toEqual([{ icon: "zap", label: "Purge everything" }]);
  });

  it("shows URL count when mode is files", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({
          configuration: {
            mode: "files",
            files: ["https://example.com/a.js", "https://example.com/b.css"],
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "link", label: "2 URLs" }]);
  });

  it("shows singular URL label for single file", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "files", files: ["https://example.com/a.js"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "link", label: "1 URL" }]);
  });

  it("shows tag count when mode is tags", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "tags", tags: ["v1.2.3", "static-assets"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "tag", label: "2 tags" }]);
  });

  it("shows host count when mode is hosts", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "hosts", hosts: ["preview.example.com"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "server", label: "1 host" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = purgeCacheMapper.props(buildPropsCtx({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("purgeCacheMapper.getExecutionDetails", () => {
  it("returns details with mode, zone and purge ID", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              id: "purge-abc",
              mode: "files",
              files: ["https://example.com/a.js", "https://example.com/b.js"],
            }),
          ],
        },
      },
    });
    const details = purgeCacheMapper.getExecutionDetails(ctx);
    expect(details["Mode"]).toBe("files");
    expect(details["Zone ID"]).toBe("zone123");
    expect(details["Purge ID"]).toBe("purge-abc");
    expect(details["Files"]).toBe("2");
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => purgeCacheMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ zoneId: "z", id: "p", mode: "everything" })] } },
    });
    expect(purgeCacheMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.purgeCache", () => {
  it("maps finished success to purged", () => {
    expect(eventStateRegistry.purgeCache.getState(buildExecution())).toBe("purged");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.purgeCache.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.purgeCache.getState(failed)).toBe("failed");
  });
});
