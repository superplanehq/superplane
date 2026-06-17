import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getRepositoryMapper } from "./get_repository";
import { getPackageStatusMapper } from "./get_package_status";
import { getPackageMapper } from "./get_package";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getRepository: getRepositoryMapper,
  getPackageStatus: getPackageStatusMapper,
  getPackage: getPackageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getRepository: buildActionStateRegistry("fetched"),
  getPackageStatus: buildActionStateRegistry("retrieved"),
  getPackage: buildActionStateRegistry("fetched"),
};
