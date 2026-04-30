import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createComputeInstanceMapper } from "./create_compute_instance";
import { createImageMapper } from "./create_image";
import { deleteInstanceMapper } from "./delete_instance";
import { deleteImageMapper } from "./delete_image";
import { getInstanceMapper } from "./get_instance";
import { getImageMapper } from "./get_image";
import { manageInstancePowerMapper } from "./manage_instance_power";
import { onComputeInstanceCreatedTriggerRenderer } from "./on_compute_instance_created";
import { onInstanceStateChangeTriggerRenderer } from "./on_instance_state_change";
import { updateInstanceMapper } from "./update_instance";
import { buildActionStateRegistry } from "../utils";
import { updateImageMapper } from "./update_image";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createComputeInstance: createComputeInstanceMapper,
  getInstance: getInstanceMapper,
  updateInstance: updateInstanceMapper,
  manageInstancePower: manageInstancePowerMapper,
  deleteInstance: deleteInstanceMapper,
  createImage: createImageMapper,
  getImage: getImageMapper,
  updateImage: updateImageMapper,
  deleteImage: deleteImageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComputeInstanceCreated: onComputeInstanceCreatedTriggerRenderer,
  onInstanceStateChange: onInstanceStateChangeTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createComputeInstance: buildActionStateRegistry("created"),
  getInstance: buildActionStateRegistry("fetched"),
  updateInstance: buildActionStateRegistry("updated"),
  manageInstancePower: buildActionStateRegistry("completed"),
  deleteInstance: buildActionStateRegistry("deleted"),
  createImage: buildActionStateRegistry("created"),
  getImage: buildActionStateRegistry("retrieved"),
  updateImage: buildActionStateRegistry("updated"),
  deleteImage: buildActionStateRegistry("deleted"),
};
