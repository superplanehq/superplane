import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { triggerDeployMapper, DEPLOY_STATE_REGISTRY } from "./trigger_deploy";
import { onDeploymentTriggerRenderer } from "./on_deployment";
import { getDeploymentMapper } from "./get_deployment";
import { rollbackDeployMapper } from "./rollback_deploy";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerDeploy: triggerDeployMapper,
  getDeployment: getDeploymentMapper,
  rollbackDeploy: rollbackDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeployment: onDeploymentTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerDeploy: DEPLOY_STATE_REGISTRY,
};
