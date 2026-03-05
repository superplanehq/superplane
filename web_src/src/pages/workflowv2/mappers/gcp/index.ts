import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { publishMessageMapper } from "./publish_message";
import { onTopicMessageTriggerRenderer } from "./on_topic_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
  "pubsub.publishMessage": publishMessageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
  "pubsub.onTopicMessage": onTopicMessageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  "pubsub.publishMessage": buildActionStateRegistry("published"),
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
