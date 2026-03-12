import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper, cloudBuildBaseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { CLOUD_BUILD_EXECUTION_STATE_REGISTRY } from "./cloudbuild";
import { onVMInstanceTriggerRenderer } from "./on_vm_instance";
import { onBuildCompleteTriggerRenderer } from "./on_build_complete";
import { onArtifactPushTriggerRenderer } from "./on_artifact_push";
import { onArtifactAnalysisTriggerRenderer } from "./on_artifact_analysis";
import { runTriggerMapper } from "./run_trigger";
import { invokeFunctionMapper } from "./invoke_function";
import { getArtifactMapper, getArtifactAnalysisMapper } from "./artifact_registry_mapper";
import {
  publishMessageMapper,
  createTopicMapper,
  deleteTopicMapper,
  createSubscriptionMapper,
  deleteSubscriptionMapper,
  PUBSUB_ACTION_STATE_REGISTRY,
} from "./pubsub_mapper";
import { onMessageTriggerRenderer } from "./on_message";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createVM: baseMapper,
  "cloudbuild.createBuild": cloudBuildBaseMapper,
  "cloudbuild.getBuild": cloudBuildBaseMapper,
  "cloudbuild.runTrigger": runTriggerMapper,
  "cloudfunctions.invokeFunction": invokeFunctionMapper,
  "artifactregistry.getArtifact": getArtifactMapper,
  "artifactregistry.getArtifactAnalysis": getArtifactAnalysisMapper,
  "pubsub.publishMessage": publishMessageMapper,
  "pubsub.createTopic": createTopicMapper,
  "pubsub.deleteTopic": deleteTopicMapper,
  "pubsub.createSubscription": createSubscriptionMapper,
  "pubsub.deleteSubscription": deleteSubscriptionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onVMInstance: onVMInstanceTriggerRenderer,
  "cloudbuild.onBuildComplete": onBuildCompleteTriggerRenderer,
  "artifactregistry.onArtifactPush": onArtifactPushTriggerRenderer,
  "artifactregistry.onArtifactAnalysis": onArtifactAnalysisTriggerRenderer,
  "pubsub.onMessage": onMessageTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createVM: buildActionStateRegistry("completed"),
  "cloudbuild.createBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.getBuild": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudbuild.runTrigger": CLOUD_BUILD_EXECUTION_STATE_REGISTRY,
  "cloudfunctions.invokeFunction": buildActionStateRegistry("completed"),
  "artifactregistry.getArtifact": buildActionStateRegistry("completed"),
  "artifactregistry.getArtifactAnalysis": buildActionStateRegistry("completed"),
  "pubsub.publishMessage": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.createTopic": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.deleteTopic": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.createSubscription": PUBSUB_ACTION_STATE_REGISTRY,
  "pubsub.deleteSubscription": PUBSUB_ACTION_STATE_REGISTRY,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {};
