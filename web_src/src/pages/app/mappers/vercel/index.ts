import type { ComponentBaseMapper, TriggerRenderer } from "../types";
import { deployMapper } from "./deploy";
import { getDeploymentMapper } from "./get_deployment";
import { onDeploymentTriggerRenderer } from "./common";
import { listDeploymentsMapper } from "./list_deployments";
import { cancelDeploymentMapper, rollbackMapper } from "./deployment_ops";
import {
  addDomainMapper,
  createProjectMapper,
  getProjectMapper,
  removeDomainMapper,
  upsertEnvVarMapper,
} from "./project_actions";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  deploy: deployMapper,
  getDeployment: getDeploymentMapper,
  listDeployments: listDeploymentsMapper,
  cancelDeployment: cancelDeploymentMapper,
  rollback: rollbackMapper,
  getProject: getProjectMapper,
  createProject: createProjectMapper,
  upsertEnvVar: upsertEnvVarMapper,
  addDomain: addDomainMapper,
  removeDomain: removeDomainMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeployment: onDeploymentTriggerRenderer,
};
