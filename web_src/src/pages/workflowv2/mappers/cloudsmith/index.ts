import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { getPackageMapper } from "./get_package";
import { onPackageEventTriggerRenderer, onPackageEventCustomFieldRenderer } from "./on_package_event";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getPackage: getPackageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPackageEvent: onPackageEventTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onPackageEvent: onPackageEventCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getPackage: buildActionStateRegistry("retrieved"),
};
