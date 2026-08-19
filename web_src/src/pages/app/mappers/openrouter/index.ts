import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { getCreditsMapper } from "./get_credits";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  chatCompletion: baseMapper,
  getCredits: getCreditsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  chatCompletion: buildActionStateRegistry("completed"),
  getCredits: buildActionStateRegistry("completed"),
};
