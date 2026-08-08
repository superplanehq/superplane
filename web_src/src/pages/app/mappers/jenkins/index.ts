import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { triggerBuildMapper } from "./trigger_build";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerBuild: triggerBuildMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};