import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { jfrogArtifactoryBaseMapper } from "./base";
import { DEFAULT_STATE_REGISTRY } from "../stateRegistry";
import { onArtifactUploadedTriggerRenderer } from "./on_artifact_uploaded";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  getArtifactInfo: jfrogArtifactoryBaseMapper,
  deleteArtifact: jfrogArtifactoryBaseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onArtifactUploaded: onArtifactUploadedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  getArtifactInfo: DEFAULT_STATE_REGISTRY,
  deleteArtifact: DEFAULT_STATE_REGISTRY,
};
