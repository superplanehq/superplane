import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { deployMapper, DEPLOY_STATE_REGISTRY } from "./deploy";
import { cancelDeployMapper } from "./cancel_deploy";
import { getDeployMapper } from "./get_deploy";
import { getServiceMapper } from "./get_service";
import { onBuildTriggerRenderer } from "./on_build";
import { onDeployTriggerRenderer } from "./on_deploy";
import { rollbackDeployMapper } from "./rollback_deploy";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  deploy: deployMapper,
  getService: getServiceMapper,
  getDeploy: getDeployMapper,
  cancelDeploy: cancelDeployMapper,
  rollbackDeploy: rollbackDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeploy: onDeployTriggerRenderer,
  onBuild: onBuildTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  deploy: DEPLOY_STATE_REGISTRY,
  cancelDeploy: DEPLOY_STATE_REGISTRY,
  rollbackDeploy: DEPLOY_STATE_REGISTRY,
};
