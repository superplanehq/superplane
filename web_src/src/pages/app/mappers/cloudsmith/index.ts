import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getRepositoryMapper } from "./get_repository";
import { getPackageComplianceMapper } from "./get_package_compliance";
import { onComplianceCheckCompletedTriggerRenderer } from "./on_compliance_check_completed";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getRepository: getRepositoryMapper,
  getPackageCompliance: getPackageComplianceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComplianceCheckCompleted: onComplianceCheckCompletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getRepository: buildActionStateRegistry("fetched"),
  getPackageCompliance: buildActionStateRegistry("fetched"),
  onComplianceCheckCompleted: buildActionStateRegistry("triggered"),
};
