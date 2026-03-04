import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper, cloudBuildBaseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { CLOUD_BUILD_EXECUTION_STATE_REGISTRY } from "./cloudbuild";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { onBuildCompleteTriggerRenderer } from "./on_build_complete";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
  "cloudbuild.createBuild": cloudBuildBaseMapper,
  "cloudbuild.getBuild": cloudBuildBaseMapper,
  "cloudbuild.runTrigger": cloudBuildBaseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
  "cloudbuild.onBuildComplete": onBuildCompleteTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  "cloudbuild.createBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.getBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.runTrigger": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
