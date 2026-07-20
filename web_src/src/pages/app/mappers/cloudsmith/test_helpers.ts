import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import type { PackageData, PackageOperationResult, RepositoryData, VulnerabilityPolicyData } from "./types";

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
    self_webapp_url: "https://cloudsmith.io/~acme/repos/production/",
    ...overrides,
  };
}

export function buildPackageData(overrides?: Partial<PackageData>): PackageData {
  return {
    slug: "my-package-1-0-0",
    slug_perm: "perm123abc456",
    name: "my-package",
    version: "1.0.0",
    format: "python",
    repository: "production",
    namespace: "acme",
    uploaded_at: "2026-01-15T10:00:00.000Z",
    status: 2,
    status_str: "Available",
    status_updated_at: "2026-01-15T10:00:05.000Z",
    stage: 9,
    stage_str: "Fully Synchronised",
    stage_updated_at: "2026-01-15T10:00:05.000Z",
    is_sync_awaiting: false,
    is_sync_completed: true,
    is_sync_failed: false,
    is_sync_in_flight: false,
    is_sync_in_progress: false,
    sync_finished_at: "2026-01-15T10:00:05.000Z",
    sync_progress: 100,
    is_quarantined: false,
    policy_violated: false,
    security_scan_status: "No Vulnerabilities Found",
    checksum_md5: "d41d8cd98f00b204e9800998ecf8427e",
    checksum_sha256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    size: 524288,
    size_str: "512.0 KB",
    cdn_url: "https://dl.cloudsmith.io/public/acme/production/python/my-package-1.0.0.tar.gz",
    self_html_url: "https://cloudsmith.io/~acme/repos/production/packages/detail/python/my-package/1.0.0/",
    self_webapp_url: "https://app.cloudsmith.com/acme/r/production/python/my-package/1.0.0/",
    ...overrides,
  };
}

export function buildPackageOutput(data: unknown, type = "cloudsmith.package.details"): OutputPayload {
  return {
    type,
    timestamp: new Date().toISOString(),
    data,
  };
}

export function buildPackageOperationResult(overrides?: Partial<PackageOperationResult>): PackageOperationResult {
  return {
    repository: "acme/production",
    package: "pkg_123",
    data: {
      name: "billing-api",
      display_name: "billing-api",
      slug_perm: "pkg_123",
      version: "1.2.3",
      format: "docker",
      status_str: "Completed",
      self_webapp_url: "https://cloudsmith.io/~acme/repos/production/packages/detail/docker/pkg_123/",
      tags: {
        latest: true,
        production: true,
        inactive: false,
      },
    },
    ...overrides,
  };
}

export function buildVulnerabilityPolicyData(overrides?: Partial<VulnerabilityPolicyData>): VulnerabilityPolicyData {
  return {
    name: "Block critical vulnerabilities",
    description: "Quarantine packages with critical vulnerabilities",
    min_severity: "Critical",
    package_query_string: "format:docker",
    on_violation_quarantine: true,
    allow_unknown_severity: false,
    slug_perm: "abc123def456",
    created_at: "2026-01-15T10:00:00.000Z",
    updated_at: "2026-01-20T14:30:00.000Z",
    ...overrides,
  };
}
