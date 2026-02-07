import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onBuildTriggerRenderer } from "./on_build";
import { onDeployTriggerRenderer } from "./on_deploy";
import { triggerDeployMapper } from "./trigger_deploy";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerDeploy: triggerDeployMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onDeploy: onDeployTriggerRenderer,
  onBuild: onBuildTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerDeploy: buildActionStateRegistry("triggered"),
};
