import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getRepositoryMapper } from "./get_repository";
import { getPackageMapper } from "./get_package";
import { listPackagesMapper } from "./list_packages";
import { promotePackageMapper, promotePackageEventStateRegistry } from "./promote_package";
import { scanPackageMapper } from "./scan_package";
import { quarantinePackageMapper, QUARANTINE_PACKAGE_STATE_REGISTRY } from "./quarantine_package";
import { getPackageVulnerabilitiesMapper } from "./get_package_vulnerabilities";
import { createVulnerabilityPolicyMapper } from "./create_vulnerability_policy";
import { getVulnerabilityPolicyMapper } from "./get_vulnerability_policy";
import { deleteVulnerabilityPolicyMapper } from "./delete_vulnerability_policy";
import { onSecurityScanCompletedTriggerRenderer } from "./on_security_scan_completed";
import { onPackageCreatedTriggerRenderer } from "./on_package_created";
import { resyncPackageMapper } from "./resync_package";
import { tagPackageMapper } from "./tag_package";
import { deletePackageMapper } from "./delete_package";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getRepository: getRepositoryMapper,
  getPackage: getPackageMapper,
  resyncPackage: resyncPackageMapper,
  tagPackage: tagPackageMapper,
  deletePackage: deletePackageMapper,
  listPackages: listPackagesMapper,
  promotePackage: promotePackageMapper,
  scanPackage: scanPackageMapper,
  quarantinePackage: quarantinePackageMapper,
  getPackageVulnerabilities: getPackageVulnerabilitiesMapper,
  createVulnerabilityPolicy: createVulnerabilityPolicyMapper,
  getVulnerabilityPolicy: getVulnerabilityPolicyMapper,
  deleteVulnerabilityPolicy: deleteVulnerabilityPolicyMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onSecurityScanCompleted: onSecurityScanCompletedTriggerRenderer,
  onPackageCreated: onPackageCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getRepository: buildActionStateRegistry("fetched"),
  getPackage: buildActionStateRegistry("fetched"),
  listPackages: buildActionStateRegistry("completed"),
  promotePackage: promotePackageEventStateRegistry,
  createVulnerabilityPolicy: buildActionStateRegistry("created"),
  getVulnerabilityPolicy: buildActionStateRegistry("fetched"),
  deleteVulnerabilityPolicy: buildActionStateRegistry("deleted"),
  onSecurityScanCompleted: buildActionStateRegistry("triggered"),
  onPackageCreated: buildActionStateRegistry("triggered"),
  resyncPackage: buildActionStateRegistry("resynced"),
  tagPackage: buildActionStateRegistry("tagged"),
  deletePackage: buildActionStateRegistry("deleted"),
  scanPackage: buildActionStateRegistry("scheduled"),
  quarantinePackage: QUARANTINE_PACKAGE_STATE_REGISTRY,
  getPackageVulnerabilities: buildActionStateRegistry("fetched"),
};
