import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPageAddedTriggerRenderer } from "./on_page_added";

export const componentMappers: Record<string, ComponentBaseMapper> = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPageAdded: onPageAddedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
