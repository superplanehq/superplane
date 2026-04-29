import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createComputeInstanceMapper } from "./create_compute_instance";
import { createImageMapper } from "./create_image";
import { deleteImageMapper } from "./delete_image";
import { getImageMapper } from "./get_image";
import { onComputeInstanceCreatedTriggerRenderer } from "./on_compute_instance_created";
import { buildActionStateRegistry } from "../utils";
import { updateImageMapper } from "./update_image";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createComputeInstance: createComputeInstanceMapper,
  createImage: createImageMapper,
  getImage: getImageMapper,
  updateImage: updateImageMapper,
  deleteImage: deleteImageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComputeInstanceCreated: onComputeInstanceCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createComputeInstance: buildActionStateRegistry("created"),
  createImage: buildActionStateRegistry("created"),
  getImage: buildActionStateRegistry("retrieved"),
  updateImage: buildActionStateRegistry("updated"),
  deleteImage: buildActionStateRegistry("deleted"),
};
