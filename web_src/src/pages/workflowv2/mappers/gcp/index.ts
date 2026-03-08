import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper, cloudBuildBaseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { CLOUD_BUILD_EXECUTION_STATE_REGISTRY } from "./cloudbuild";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { onBuildCompleteTriggerRenderer } from "./on_build_complete";
import { onObjectFinalizedTriggerRenderer } from "./on_object_finalized";
import { runTriggerMapper } from "./run_trigger";
import { invokeFunctionMapper } from "./invoke_function";
import { getObjectMapper } from "./get_object";
import { uploadObjectMapper } from "./upload_object";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
  "cloudbuild.createBuild": cloudBuildBaseMapper,
  "cloudbuild.getBuild": cloudBuildBaseMapper,
  "cloudbuild.runTrigger": runTriggerMapper,
  "cloudfunctions.invokeFunction": invokeFunctionMapper,
  "cloudstorage.getObject": getObjectMapper,
  "cloudstorage.uploadObject": uploadObjectMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
  "cloudbuild.onBuildComplete": onBuildCompleteTriggerRenderer,
  "cloudstorage.onObjectFinalized": onObjectFinalizedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  "cloudbuild.createBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.getBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.runTrigger": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudfunctions.invokeFunction": buildActionStateRegistry("completed"),
  "cloudstorage.getObject": buildActionStateRegistry("completed"),
  "cloudstorage.uploadObject": buildActionStateRegistry("completed"),
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
