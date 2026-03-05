import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { invokeFunctionMapper } from "./invoke_function";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
  "cloudfunctions.invokeFunction": invokeFunctionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  "cloudfunctions.invokeFunction": buildActionStateRegistry("completed"),
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
