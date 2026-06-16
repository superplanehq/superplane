import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import type { RepositoryData } from "./types";

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get Repository",
    componentName: "cloudsmith.getRepository",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudsmith.repository.fetched",
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

export function buildRepositoryData(overrides?: Partial<RepositoryData>): RepositoryData {
  return {
    name: "Production",
    slug: "production",
    namespace: "acme",
    repository_type_str: "Private",
    is_private: true,
    storage_region: "us-ohio",
    size: 524288000,
    size_str: "500.0 MB",
    package_count: 312,
    num_downloads: 18234,
    num_quarantined_packages: 1,
    num_policy_violated_packages: 2,
    self_html_url: "https://cloudsmith.io/~acme/repos/production/",
    ...overrides,
  };
}
