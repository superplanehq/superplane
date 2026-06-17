import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getRepositoryMapper } from "./get_repository";
import { getPackageMapper } from "./get_package";
import { deletePackageMapper, resyncPackageMapper, tagPackageMapper } from "./package_operations";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getRepository: getRepositoryMapper,
  getPackage: getPackageMapper,
  resyncPackage: resyncPackageMapper,
  tagPackage: tagPackageMapper,
  deletePackage: deletePackageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getRepository: buildActionStateRegistry("fetched"),
  getPackage: buildActionStateRegistry("fetched"),
  resyncPackage: buildActionStateRegistry("resynced"),
  tagPackage: buildActionStateRegistry("tagged"),
  deletePackage: buildActionStateRegistry("deleted"),
};
