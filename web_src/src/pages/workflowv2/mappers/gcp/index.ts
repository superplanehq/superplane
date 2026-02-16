import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onVMCreatedTriggerRenderer } from "./on_vm_created";
import { onVMCreatedCustomFieldRenderer } from "./on_vm_created_field_renderer";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMCreated: onVMCreatedTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onVMCreated: onVMCreatedCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
};
