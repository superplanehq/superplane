import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { deployMapper, DEPLOY_STATE_REGISTRY } from "./deploy";
import { cancelDeployMapper } from "./cancel_deploy";
import { getDeployMapper } from "./get_deploy";
import { getServiceMapper } from "./get_service";
import { onBuildTriggerRenderer } from "./on_build";
import { onDeployTriggerRenderer } from "./on_deploy";
import { PURGE_CACHE_STATE_REGISTRY, purgeCacheMapper } from "./purge_cache";
import { rollbackDeployMapper } from "./rollback_deploy";
import { updateEnvVarMapper } from "./update_env_var";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  deploy: deployMapper,
  getService: getServiceMapper,
  getDeploy: getDeployMapper,
  cancelDeploy: cancelDeployMapper,
  rollbackDeploy: rollbackDeployMapper,
  purgeCache: purgeCacheMapper,
  updateEnvVar: updateEnvVarMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeploy: onDeployTriggerRenderer,
  onBuild: onBuildTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  deploy: DEPLOY_STATE_REGISTRY,
  cancelDeploy: DEPLOY_STATE_REGISTRY,
  rollbackDeploy: DEPLOY_STATE_REGISTRY,
  purgeCache: PURGE_CACHE_STATE_REGISTRY,
};
