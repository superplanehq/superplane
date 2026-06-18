import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getRepositoryMapper } from "./get_repository";
import { onComplianceCheckCompletedTriggerRenderer } from "./on_compliance_check_completed";
import { onPackageCreatedTriggerRenderer } from "./on_package_created";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getRepository: getRepositoryMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComplianceCheckCompleted: onComplianceCheckCompletedTriggerRenderer,
  onPackageCreated: onPackageCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getRepository: buildActionStateRegistry("fetched"),
  onComplianceCheckCompleted: buildActionStateRegistry("triggered"),
  onPackageCreated: buildActionStateRegistry("triggered"),
};
