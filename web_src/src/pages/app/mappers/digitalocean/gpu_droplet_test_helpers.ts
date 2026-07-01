import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test GPU Node",
    componentName: "digitalocean.createGPUDroplet",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildOutput(data: unknown): OutputPayload {
  return {
    type: "digitalocean.result",
    timestamp: new Date().toISOString(),
    data,
  };
}

export function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
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

export function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

export function buildDropletData(overrides?: Record<string, unknown>) {
  return {
    id: 123456,
    name: "gpu-droplet-1",
    status: "active",
    size_slug: "gpu-h100x1-80gb",
    memory: 245760,
    vcpus: 20,
    disk: 480,
    image: { id: 1, name: "Ubuntu 22.04 (LTS) x64", slug: "ubuntu-22-04-x64" },
    region: { name: "New York 3", slug: "nyc3" },
    networks: {
      v4: [{ type: "public", ip_address: "1.2.3.4" }],
    },
    tags: ["gpu", "ml"],
    ...overrides,
  };
}
