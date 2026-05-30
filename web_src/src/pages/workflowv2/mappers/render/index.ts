import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { deployMapper, DEPLOY_STATE_REGISTRY } from "./deploy";
import { cancelDeployMapper } from "./cancel_deploy";
import { getDeployMapper } from "./get_deploy";
import { getServiceMapper } from "./get_service";
import { onBuildTriggerRenderer } from "./on_build";
import { onDeployTriggerRenderer } from "./on_deploy";
import { PURGE_CACHE_STATE_REGISTRY, purgeCacheMapper } from "./purge_cache";
import { rollbackDeployMapper } from "./rollback_deploy";
import { SCALE_SERVICE_STATE_REGISTRY, scaleServiceMapper } from "./scale_service";
import { updateEnvVarMapper } from "./update_env_var";
import { addCustomDomainMapper } from "./add_custom_domain";
import { removeCustomDomainMapper } from "./remove_custom_domain";
import { renderOperationMapper } from "./operations";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  deploy: deployMapper,
  getService: getServiceMapper,
  getDeploy: getDeployMapper,
  listDeploys: renderOperationMapper,
  cancelDeploy: cancelDeployMapper,
  rollbackDeploy: rollbackDeployMapper,
  purgeCache: purgeCacheMapper,
  scaleService: scaleServiceMapper,
  updateEnvVar: updateEnvVarMapper,
  "service.addCustomDomain": addCustomDomainMapper,
  "service.removeCustomDomain": removeCustomDomainMapper,
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
  scaleService: SCALE_SERVICE_STATE_REGISTRY,
};
