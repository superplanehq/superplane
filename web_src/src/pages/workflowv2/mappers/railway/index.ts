import {
  ComponentBaseMapper,
  TriggerRenderer,
  CustomFieldRenderer,
  EventStateRegistry,
  ExecutionInfo,
  StateFunction,
} from "../types";
import { onDeploymentEventTriggerRenderer } from "./on_deployment_event";
import { triggerDeployMapper } from "./trigger_deploy";
import { onDeploymentEventCustomFieldRenderer } from "./custom_field_renderer";
import { DEFAULT_EVENT_STATE_MAP, EventState, EventStateMap } from "@/ui/componentBase";
import { defaultStateFunction } from "../stateRegistry";

export const TRIGGER_DEPLOY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const triggerDeployStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  // Check for failed output
  const outputs = execution.outputs as { failed?: { data?: unknown }[] } | undefined;
  if (outputs?.failed?.length) {
    return "failed";
  }

  return defaultStateFunction(execution);
};

export const TRIGGER_DEPLOY_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TRIGGER_DEPLOY_STATE_MAP,
  getState: triggerDeployStateFunction,
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerDeploy: triggerDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeploymentEvent: onDeploymentEventTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onDeploymentEvent: onDeploymentEventCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerDeploy: TRIGGER_DEPLOY_STATE_REGISTRY,
};
