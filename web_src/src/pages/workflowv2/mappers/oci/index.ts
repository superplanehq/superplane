import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createComputeInstanceMapper } from "./create_compute_instance";
import { deleteInstanceMapper } from "./delete_instance";
import { getInstanceMapper } from "./get_instance";
import { manageInstancePowerMapper } from "./manage_instance_power";
import { onComputeInstanceCreatedTriggerRenderer } from "./on_compute_instance_created";
import { createApplicationMapper } from "./create_application";
import { deleteApplicationMapper } from "./delete_application";
import { createFunctionMapper } from "./create_function";
import { deleteFunctionMapper } from "./delete_function";
import { invokeFunctionMapper } from "./invoke_function";
import { onInstanceStateChangeTriggerRenderer } from "./on_instance_state_change";
import { updateInstanceMapper } from "./update_instance";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createComputeInstance: createComputeInstanceMapper,
  createApplication: createApplicationMapper,
  deleteApplication: deleteApplicationMapper,
  createFunction: createFunctionMapper,
  deleteFunction: deleteFunctionMapper,
  invokeFunction: invokeFunctionMapper,
  getInstance: getInstanceMapper,
  updateInstance: updateInstanceMapper,
  manageInstancePower: manageInstancePowerMapper,
  deleteInstance: deleteInstanceMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onComputeInstanceCreated: onComputeInstanceCreatedTriggerRenderer,
  onInstanceStateChange: onInstanceStateChangeTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createComputeInstance: buildActionStateRegistry("created"),
  createApplication: buildActionStateRegistry("created"),
  deleteApplication: buildActionStateRegistry("deleted"),
  createFunction: buildActionStateRegistry("created"),
  deleteFunction: buildActionStateRegistry("deleted"),
  invokeFunction: buildActionStateRegistry("success"),
  getInstance: buildActionStateRegistry("fetched"),
  updateInstance: buildActionStateRegistry("updated"),
  manageInstancePower: buildActionStateRegistry("completed"),
  deleteInstance: buildActionStateRegistry("deleted"),
};
