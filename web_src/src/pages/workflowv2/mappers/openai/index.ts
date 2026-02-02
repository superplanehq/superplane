import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../github/utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  textPrompt: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  textPrompt: buildActionStateRegistry("completed"),
};
