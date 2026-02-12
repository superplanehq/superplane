import { ComponentBaseMapper, TriggerRenderer, CustomFieldRenderer, EventStateRegistry } from "../types";
import { onDeploymentEventTriggerRenderer } from "./on_deployment_event";
import {
  triggerDeployMapper,
  TRIGGER_DEPLOY_STATE_MAP,
  triggerDeployStateFunction,
  TRIGGER_DEPLOY_STATE_REGISTRY,
} from "./trigger_deploy";
import { onDeploymentEventCustomFieldRenderer } from "./custom_field_renderer";

export { TRIGGER_DEPLOY_STATE_MAP, triggerDeployStateFunction, TRIGGER_DEPLOY_STATE_REGISTRY };

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
