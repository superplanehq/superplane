import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { triggerDeployMapper, DEPLOY_STATE_REGISTRY } from "./trigger_deploy";
import { onDeploymentTriggerRenderer } from "./on_deployment";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerDeploy: triggerDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeployment: onDeploymentTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerDeploy: DEPLOY_STATE_REGISTRY,
};
