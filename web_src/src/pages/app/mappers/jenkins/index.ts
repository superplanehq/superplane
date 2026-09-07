import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { triggerBuildMapper } from "./trigger_build";
import { getBuildMapper } from "./get_build";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerBuild: triggerBuildMapper,
  getBuild: getBuildMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};