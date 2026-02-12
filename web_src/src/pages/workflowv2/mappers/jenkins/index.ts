import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import {
  TRIGGER_BUILD_STATE_REGISTRY,
  triggerBuildCustomFieldRenderer,
  triggerBuildMapper,
} from "./trigger_build";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerBuild: triggerBuildMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerBuild: TRIGGER_BUILD_STATE_REGISTRY,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  triggerBuild: triggerBuildCustomFieldRenderer,
};
