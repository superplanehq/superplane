import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onBlobCreatedTriggerRenderer } from "./on_blob_created";
import { onBlobDeletedTriggerRenderer } from "./on_blob_deleted";
import { onImagePushedTriggerRenderer } from "./on_image_pushed";
import { onImageDeletedTriggerRenderer } from "./on_image_deleted";
import { invokeFunctionMapper } from "./invoke_function";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  invokeFunction: invokeFunctionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onBlobCreated: onBlobCreatedTriggerRenderer,
  onBlobDeleted: onBlobDeletedTriggerRenderer,
  onContainerImagePushed: onImagePushedTriggerRenderer,
  onContainerImageDeleted: onImageDeletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  deleteVirtualMachine: buildActionStateRegistry("deleted"),
};
