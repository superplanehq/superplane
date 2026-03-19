import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onBlobCreatedTriggerRenderer } from "./on_blob_created";
import { onBlobDeletedTriggerRenderer } from "./on_blob_deleted";
import { onImagePushedTriggerRenderer } from "./on_image_pushed";
import { onImageDeletedTriggerRenderer } from "./on_image_deleted";
import {
  createServiceBusQueueMapper,
  deleteServiceBusQueueMapper,
  getServiceBusQueueMapper,
  sendServiceBusMessageMapper,
  createServiceBusTopicMapper,
  deleteServiceBusTopicMapper,
  getServiceBusTopicMapper,
  publishServiceBusMessageMapper,
} from "./servicebus_components";
import {
  onServiceBusMessageAvailableTriggerRenderer,
  onServiceBusDeadLetterAvailableTriggerRenderer,
  onServiceBusMessageReceivedTriggerRenderer,
} from "./servicebus_triggers";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createServiceBusQueue: createServiceBusQueueMapper,
  deleteServiceBusQueue: deleteServiceBusQueueMapper,
  getServiceBusQueue: getServiceBusQueueMapper,
  sendServiceBusMessage: sendServiceBusMessageMapper,
  createServiceBusTopic: createServiceBusTopicMapper,
  deleteServiceBusTopic: deleteServiceBusTopicMapper,
  getServiceBusTopic: getServiceBusTopicMapper,
  publishServiceBusMessage: publishServiceBusMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onBlobCreated: onBlobCreatedTriggerRenderer,
  onBlobDeleted: onBlobDeletedTriggerRenderer,
  onContainerImagePushed: onImagePushedTriggerRenderer,
  onContainerImageDeleted: onImageDeletedTriggerRenderer,
  onServiceBusMessageAvailable: onServiceBusMessageAvailableTriggerRenderer,
  onServiceBusDeadLetterAvailable: onServiceBusDeadLetterAvailableTriggerRenderer,
  onServiceBusMessageReceived: onServiceBusMessageReceivedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  deleteVirtualMachine: buildActionStateRegistry("deleted"),
  deleteServiceBusQueue: buildActionStateRegistry("deleted"),
  deleteServiceBusTopic: buildActionStateRegistry("deleted"),
};
