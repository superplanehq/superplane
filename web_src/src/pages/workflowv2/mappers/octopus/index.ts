import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { deployReleaseMapper, DEPLOY_RELEASE_STATE_REGISTRY } from "./deploy_release";
import { onDeploymentEventTriggerRenderer } from "./on_deployment_event";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  deployRelease: deployReleaseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeploymentEvent: onDeploymentEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  deployRelease: DEPLOY_RELEASE_STATE_REGISTRY,
};
